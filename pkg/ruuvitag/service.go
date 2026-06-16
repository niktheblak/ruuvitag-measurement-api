package ruuvitag

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
)

type Config struct {
	ConnString string
	Table      string
	NameTable  string
	NameColumn string
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
	dbpool       *pgxpool.Pool
	columnMap    map[string]string
	columnValues map[string]interface{}
	qb           *QueryBuilder
	logger       *slog.Logger
}

// New creates a new instance of the service using the given config
func New(ctx context.Context, cfg Config) (Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	cfg.Logger.LogAttrs(ctx, slog.LevelDebug, "Columns", slog.Any("column_map", cfg.Columns))
	dbpool, err := pgxpool.New(ctx, cfg.ConnString)
	if err != nil {
		return nil, err
	}
	s := &service{
		dbpool:    dbpool,
		columnMap: cfg.Columns,
		qb: &QueryBuilder{
			Table:      cfg.Table,
			NameTable:  cfg.NameTable,
			NameColumn: cfg.NameColumn,
			Columns:    cfg.Columns,
		},
		logger: cfg.Logger,
	}
	s.columnValues = make(map[string]interface{})
	for _, c := range cfg.Columns {
		s.columnValues[c] = struct{}{}
	}
	return s, nil
}

func (s *service) validColumns(columns []string) ([]string, error) {
	if len(columns) == 0 {
		// no columns explicitly requested; return all configured columns
		for _, c := range s.columnMap {
			columns = append(columns, c)
		}
		return columns, nil
	}
	for _, c := range columns {
		_, ok := s.columnValues[c]
		if !ok {
			return nil, fmt.Errorf("%w: %s", sensor.ErrInvalidColumn, c)
		}
	}
	return columns, nil
}

func (s *service) Ping(ctx context.Context) error {
	if err := s.dbpool.Ping(ctx); err != nil {
		return err
	}
	return nil
}

func (s *service) Close() error {
	s.dbpool.Close()
	return nil
}

func (s *service) Latest(ctx context.Context, n int, columns []string, names []string) (measurements map[string][]sensor.Fields, err error) {
	columns, err = s.validColumns(columns)
	if err != nil {
		return
	}
	var macs map[string]string
	if len(names) == 0 {
		macs, err = s.queryAllNames(ctx)
		if err != nil {
			return
		}
	} else {
		macs, err = s.queryNames(ctx, names)
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Querying measurements from RuuviTags", slog.Any("names", macs))
	measurements = make(map[string][]sensor.Fields)
	for name, mac := range macs {
		var ms []sensor.Fields
		ms, err = s.queryMeasurements(ctx, columns, mac, n)
		if err != nil {
			return
		}
		measurements[name] = append(measurements[name], ms...)
	}
	return
}

func (s *service) queryAllNames(ctx context.Context) (map[string]string, error) {
	q := s.qb.AllNames()
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag all names query", slog.String("query", q))
	return s.runNamesQuery(ctx, q)
}

func (s *service) queryNames(ctx context.Context, names []string) (map[string]string, error) {
	q := s.qb.Names(names)
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag names query", slog.String("query", q))
	return s.runNamesQuery(ctx, q)
}

func (s *service) runNamesQuery(ctx context.Context, q string) (map[string]string, error) {
	rows, err := s.dbpool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	namesMap := make(map[string]string)
	for rows.Next() {
		var mac, name string
		if err = rows.Scan(&mac, &name); err != nil {
			return nil, err
		}
		namesMap[name] = mac
	}
	return namesMap, rows.Err()
}

func (s *service) queryMeasurements(ctx context.Context, columns []string, mac string, n int) ([]sensor.Fields, error) {
	q, err := s.qb.Latest(columns, n)
	if err != nil {
		return nil, err
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag values query", slog.String("mac", mac), slog.String("query", q))
	rows, err := s.dbpool.Query(ctx, q, mac)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var measurements []sensor.Fields
	for rows.Next() {
		f, err := s.qb.Collect(rows, columns)
		if err != nil {
			return nil, err
		}
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Returned RuuviTag measurement", slog.Any("measurement", sensor.FromFields(f)))
		measurements = append(measurements, f)
	}
	return measurements, rows.Err()
}
