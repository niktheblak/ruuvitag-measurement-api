package ruuvitag

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/niktheblak/ruuvitag-common/pkg/psql"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
)

var (
	ErrInvalidColumn = errors.New("invalid column")
)

type Config struct {
	PsqlInfo  string
	Table     string
	NameTable string
	Columns   map[string]string
	Logger    *slog.Logger
}

type Service interface {
	// Latest returns n latest measurements from all RuuviTags.
	Latest(ctx context.Context, n int, columns []string) (measurements map[string][]sensor.Fields, err error)
	// Ping verifies the database connection is still alive.
	Ping(ctx context.Context) error
	io.Closer
}

type service struct {
	conn      *pgx.Conn
	table     string
	nameTable string
	columnMap map[string]string
	qb        *psql.QueryBuilder
	logger    *slog.Logger
}

// New creates a new instance of the service using the given config
func New(ctx context.Context, cfg Config) (Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if err := validateConfigColumns(cfg.Columns); err != nil {
		return nil, err
	}
	cfg.Logger.LogAttrs(ctx, slog.LevelDebug, "Columns", slog.Any("column_map", cfg.Columns))
	conn, err := pgx.Connect(ctx, cfg.PsqlInfo)
	if err != nil {
		return nil, err
	}
	return &service{
		conn:      conn,
		table:     cfg.Table,
		nameTable: cfg.NameTable,
		columnMap: cfg.Columns,
		qb: &psql.QueryBuilder{
			Table:     cfg.Table,
			NameTable: cfg.NameTable,
			Columns:   cfg.Columns,
		},
		logger: cfg.Logger,
	}, nil
}

func (s *service) validColumns(columns []string) ([]string, error) {
	if len(columns) == 0 {
		// no columns explicitly requested; return all configured columns
		for _, c := range s.columnMap {
			columns = append(columns, c)
		}
	}
	if err := s.validateColumns(columns); err != nil {
		return nil, err
	}
	return columns, nil
}

func (s *service) Latest(ctx context.Context, n int, columns []string) (measurements map[string][]sensor.Fields, err error) {
	columns, err = s.validColumns(columns)
	if err != nil {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Response columns", slog.Any("columns", columns))
	q, err := s.qb.Names()
	if err != nil {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag names query", slog.String("query", q))
	rows, err := s.conn.Query(ctx, q)
	if err != nil {
		return
	}
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return
		}
		names = append(names, name)
	}
	rows.Close()
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Returned RuuviTag names", slog.Any("names", names))
	measurements = make(map[string][]sensor.Fields)
	for _, name := range names {
		var ms []sensor.Fields
		ms, err = s.queryMeasurements(ctx, columns, name, n)
		if err != nil {
			return
		}
		for _, m := range ms {
			k := popName(&m)
			measurements[k] = append(measurements[k], m)
		}
	}
	err = rows.Err()
	return
}

func (s *service) queryMeasurements(ctx context.Context, columns []string, name string, n int) (measurements []sensor.Fields, err error) {
	q, err := s.qb.Latest(columns, n)
	if err != nil {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag values query", slog.String("name", name), slog.String("query", q))
	rows, err := s.conn.Query(ctx, q, name)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var f sensor.Fields
		f, err = s.qb.Collect(rows, columns)
		if err != nil {
			return
		}
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Returned RuuviTag measurement", slog.Any("measurement", sensor.FromFields(f)))
		measurements = append(measurements, f)
	}
	err = rows.Err()
	return
}

func (s *service) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

func (s *service) Close() error {
	return s.conn.Close(context.Background())
}

func validateConfigColumns(columns map[string]string) error {
	if len(columns) == 0 {
		return fmt.Errorf("columns cannot be empty")
	}
	_, ok := columns["time"]
	if !ok {
		return fmt.Errorf("column time is required")
	}
	_, nameOK := columns["name"]
	_, macOK := columns["mac"]
	if !nameOK && !macOK {
		return fmt.Errorf("identifier column name or mac is required")
	}
	return nil
}

func (s *service) validateColumns(requestedColumns []string) error {
	if len(requestedColumns) == 0 {
		return fmt.Errorf("%w: requested columns cannot be empty", ErrInvalidColumn)
	}
	for _, column := range requestedColumns {
		ok := false
		for _, c := range s.columnMap {
			if c == column {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("%w: unknown column %s", ErrInvalidColumn, column)
		}
	}
	timeOK := false
	for _, column := range requestedColumns {
		if column == s.columnMap["time"] {
			timeOK = true
			break
		}
	}
	if !timeOK {
		return fmt.Errorf("%w: column %s is required", ErrInvalidColumn, s.columnMap["time"])
	}
	nameOK := false
	for _, column := range requestedColumns {
		if column == s.columnMap["name"] {
			nameOK = true
			break
		}
	}
	macOK := false
	for _, column := range requestedColumns {
		if column == s.columnMap["mac"] {
			macOK = true
			break
		}
	}
	if !nameOK && !macOK {
		return fmt.Errorf("%w: identifier column %s or %s is required", ErrInvalidColumn, s.columnMap["name"], s.columnMap["mac"])
	}
	return nil
}

func popName(f *sensor.Fields) string {
	var name string
	if f.Name != nil && *f.Name != "" {
		name = *f.Name
		f.Name = nil
	}
	if name != "" {
		return name
	}
	if f.Addr != nil && *f.Addr != "" {
		name = *f.Addr
		f.Addr = nil
	}
	return name
}
