package measurement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	_ "github.com/lib/pq"

	"github.com/niktheblak/ruuvitag-measurement-api/pkg/psql"
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
	Current(ctx context.Context, loc *time.Location, columns []string) (measurements []psql.Data, err error)
	io.Closer
}

type service struct {
	db        *sql.DB
	table     string
	nameTable string
	columnMap map[string]string
	qb        *psql.QueryBuilder
	logger    *slog.Logger
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if err := validateConfigColumns(cfg.Columns); err != nil {
		return nil, err
	}
	cfg.Logger.LogAttrs(nil, slog.LevelDebug, "Columns", slog.Any("column_map", cfg.Columns))
	db, err := sql.Open("postgres", cfg.PsqlInfo)
	if err != nil {
		return nil, err
	}
	return &service{
		db:        db,
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

// Current returns current measurements
func (s *service) Current(ctx context.Context, loc *time.Location, columns []string) ([]psql.Data, error) {
	if len(columns) == 0 {
		// no columns explicitly requested; return all configured columns
		for _, c := range s.columnMap {
			columns = append(columns, c)
		}
	}
	if err := s.validateColumns(columns); err != nil {
		return nil, err
	}
	s.logger.LogAttrs(nil, slog.LevelDebug, "Response columns", slog.Any("columns", columns))
	q := s.qb.Build(columns)
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Rendered query", slog.String("query", q))
	res, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	var measurements []psql.Data
	for res.Next() {
		d, err := s.qb.Collect(res, columns)
		if err != nil {
			return nil, err
		}
		d.Timestamp = d.Timestamp.In(loc)
		measurements = append(measurements, d)
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Found measurement", slog.Any("data", d))
	}
	return measurements, res.Err()
}

func (s *service) Close() error {
	return s.db.Close()
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
