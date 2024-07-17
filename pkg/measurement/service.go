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

	"github.com/niktheblak/temperature-api/pkg/psql"
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
	Current(ctx context.Context, loc *time.Location, columns []string) (measurements map[string]map[string]any, err error)
	io.Closer
}

type service struct {
	db          *sql.DB
	table       string
	nameTable   string
	columnMap   map[string]string
	columnNames map[string]string
	qb          *psql.QueryBuilder
	logger      *slog.Logger
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	_, ok := cfg.Columns["time"]
	if !ok {
		return nil, fmt.Errorf("column time is required")
	}
	_, nameOK := cfg.Columns["name"]
	_, macOK := cfg.Columns["mac"]
	if !nameOK && !macOK {
		return nil, fmt.Errorf("identifier column name or mac is required")
	}
	columnNames := make(map[string]string)
	for k, v := range cfg.Columns {
		columnNames[v] = k
	}
	cfg.Logger.LogAttrs(nil, slog.LevelDebug, "Columns", slog.Any("column_map", cfg.Columns), slog.Any("column_names", columnNames))
	db, err := sql.Open("postgres", cfg.PsqlInfo)
	if err != nil {
		return nil, err
	}
	return &service{
		db:          db,
		table:       cfg.Table,
		nameTable:   cfg.NameTable,
		columnMap:   cfg.Columns,
		columnNames: columnNames,
		qb: &psql.QueryBuilder{
			Table:     cfg.Table,
			NameTable: cfg.NameTable,
			Columns:   cfg.Columns,
		},
		logger: cfg.Logger,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context, loc *time.Location, columns []string) (map[string]map[string]any, error) {
	if len(columns) == 0 {
		for c := range s.columnNames {
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
	measurements := make(map[string]psql.Data)
	for res.Next() {
		d, err := s.qb.Collect(res, columns)
		if err != nil {
			return nil, err
		}
		d.Timestamp = d.Timestamp.In(loc)
		var name string
		if d.Name != nil {
			name = *d.Name
			d.Name = nil
		} else if d.Addr != nil {
			name = *d.Addr
			d.Addr = nil
		} else {
			return nil, fmt.Errorf("column name or mac is required")
		}
		measurements[name] = d
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Found measurement", slog.Any("data", d))
	}
	if err = res.Err(); err != nil {
		return nil, err
	}
	renamed := make(map[string]map[string]any)
	for cn, d := range measurements {
		renamed[cn] = psql.RenameColumns(d, s.columnMap)
	}
	return renamed, nil
}

func (s *service) Close() error {
	return s.db.Close()
}

func (s *service) validateColumns(requestedColumns []string) error {
	if len(requestedColumns) == 0 {
		return fmt.Errorf("%w: requested columns cannot be empty", ErrInvalidColumn)
	}
	for _, column := range requestedColumns {
		if _, ok := s.columnNames[column]; !ok {
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
