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

type Config struct {
	ConnString string
	Table      string
	NameTable  string
	Columns    map[string]string
	Logger     *slog.Logger
}

type Service interface {
	// Latest returns n latest measurements from all RuuviTags.
	Latest(ctx context.Context, n int, columns []string, names []string) (measurements map[string][]sensor.Fields, err error)
	// Ping verifies the database connection is still alive.
	Ping(ctx context.Context) error
	io.Closer
}

type service struct {
	conn      *pgx.Conn
	connStr   string
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
	if err := sensor.ValidateColumnMapping(cfg.Columns); err != nil {
		return nil, err
	}
	cfg.Logger.LogAttrs(ctx, slog.LevelDebug, "Columns", slog.Any("column_map", cfg.Columns))
	s := &service{
		connStr:   cfg.ConnString,
		table:     cfg.Table,
		nameTable: cfg.NameTable,
		columnMap: cfg.Columns,
		qb: &psql.QueryBuilder{
			Table:     cfg.Table,
			NameTable: cfg.NameTable,
			Columns:   cfg.Columns,
		},
		logger: cfg.Logger,
	}
	return s, s.reconnect(ctx)
}

func (s *service) validColumns(columns []string) ([]string, error) {
	if len(columns) == 0 {
		// no columns explicitly requested; return all configured columns
		for _, c := range s.columnMap {
			columns = append(columns, c)
		}
	}
	if err := sensor.ValidateRequestedColumns(s.columnMap, columns); err != nil {
		return nil, err
	}
	return columns, nil
}

func (s *service) Latest(ctx context.Context, n int, columns []string, names []string) (measurements map[string][]sensor.Fields, err error) {
	if s.conn == nil || s.conn.IsClosed() {
		if err := s.reconnect(ctx); err != nil {
			return nil, err
		}
	}
	columns, err = s.validColumns(columns)
	if err != nil {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Response columns", slog.Any("columns", columns))
	if len(names) == 0 {
		names, err = s.queryNames(ctx)
		if err != nil {
			return
		}
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Querying measurements from RuuviTags with names", slog.Any("names", names))
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
	return
}

func (s *service) queryMeasurements(ctx context.Context, columns []string, name string, n int) (measurements []sensor.Fields, err error) {
	q, err := s.qb.Latest(columns, n)
	if err != nil {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag values query", slog.String("name", name), slog.String("query", q))
	rows, err := s.query(ctx, q, name)
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
	if s.conn == nil || s.conn.IsClosed() {
		if err := s.reconnect(ctx); err != nil {
			return err
		}
	}
	if err := s.conn.Ping(ctx); err != nil {
		return errors.Join(s.reconnect(ctx), s.conn.Ping(ctx))
	}
	return nil
}

func (s *service) Close() error {
	return s.conn.Close(context.Background())
}

func (s *service) queryNames(ctx context.Context) ([]string, error) {
	q, err := s.qb.Names()
	if err != nil {
		return nil, err
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag names query", slog.String("query", q))
	rows, err := s.query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *service) query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	if s.conn == nil || s.conn.IsClosed() {
		if err := s.reconnect(ctx); err != nil {
			return nil, fmt.Errorf("failed to reconnect: %w", err)
		}
	}
	rows, err := s.conn.Query(ctx, q, args...)
	if err == nil {
		return rows, nil
	}
	if err.Error() == "conn closed" {
		// reconnect and retry
		if reconnectErr := s.reconnect(ctx); reconnectErr != nil {
			return nil, fmt.Errorf("failed to reconnect: %w, original error: %w", reconnectErr, err)
		}
		rows, err = s.conn.Query(ctx, q, args...)
		if err == nil {
			return rows, nil
		}
		return nil, err
	} else {
		return nil, err
	}
}

func (s *service) reconnect(ctx context.Context) error {
	if s.conn != nil {
		if err := s.conn.Close(ctx); err != nil {
			s.logger.LogAttrs(ctx, slog.LevelWarn, "Error while closing connection", slog.String("error", err.Error()))
		}
		s.conn = nil
	}
	conn, err := pgx.Connect(ctx, s.connStr)
	if err != nil {
		return err
	}
	s.conn = conn
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
