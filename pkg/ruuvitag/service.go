package ruuvitag

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var QueryColumns = []string{
	"temperature",
	"humidity",
	"pressure",
	"acceleration_x",
	"acceleration_y",
	"acceleration_z",
	"movement_counter",
	"measurement_number",
	"dew_point",
	"battery_voltage",
	"wet_bulb",
}

var ErrInvalidColumn = errors.New("invalid column")

var MandatoryColumns = []string{
	"time",
	"mac",
}

var queryColumnMap map[string]struct{}

func init() {
	queryColumnMap = make(map[string]struct{})
	for _, column := range QueryColumns {
		queryColumnMap[column] = struct{}{}
	}
}

type Config struct {
	ConnString string
	Table      string
	NameTable  string
	Logger     *slog.Logger
}

type Service interface {
	// Latest returns n latest measurements from all RuuviTags.
	Latest(ctx context.Context, columns []string, names []string, count int) (measurements map[string][]Fields, err error)
	// Ping verifies the database connection is still alive.
	Ping(ctx context.Context) error
	io.Closer
}

type service struct {
	dbpool *pgxpool.Pool
	cfg    Config
	logger *slog.Logger
}

// New creates a new instance of the service using the given config
func New(ctx context.Context, cfg Config) (Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	dbpool, err := pgxpool.New(ctx, cfg.ConnString)
	if err != nil {
		return nil, err
	}
	s := &service{
		dbpool: dbpool,
		cfg:    cfg,
		logger: cfg.Logger,
	}
	return s, nil
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

func (s *service) Latest(ctx context.Context, columns []string, names []string, count int) (measurements map[string][]Fields, err error) {
	columns, err = validColumns(columns)
	if err != nil {
		return
	}
	if len(columns) == 0 {
		columns = slices.Clone(QueryColumns)
	}
	queryColumns := make([]string, 0, len(columns)+len(MandatoryColumns))
	queryColumns = append(queryColumns, columns...)
	queryColumns = append(queryColumns, MandatoryColumns...)
	var macs map[string]string
	if len(names) == 0 {
		macs, err = s.queryAllMACs(ctx)
	} else {
		macs, err = s.queryMACs(ctx, names)
	}
	if err != nil {
		return
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Querying measurements from RuuviTags", slog.Any("macs", macs))
	measurements = make(map[string][]Fields)
	for name, mac := range macs {
		var ms []Fields
		ms, err = s.queryMeasurements(ctx, queryColumns, mac, count)
		if err != nil {
			return
		}
		measurements[name] = append(measurements[name], ms...)
	}
	return
}

func (s *service) queryAllMACs(ctx context.Context) (map[string]string, error) {
	q := fmt.Sprintf(`SELECT mac, name FROM %s`, s.cfg.NameTable)
	return s.runMACsQuery(ctx, q)
}

func (s *service) queryMACs(ctx context.Context, names []string) (map[string]string, error) {
	quotedNames := make([]string, 0, len(names))
	for _, n := range names {
		quotedNames = append(quotedNames, fmt.Sprintf(`'%s'`, n))
	}
	queryNames := strings.Join(quotedNames, ",")
	q := fmt.Sprintf(`SELECT mac, name FROM %s WHERE name = ANY(ARRAY[%s])`, s.cfg.NameTable, queryNames)
	return s.runMACsQuery(ctx, q)
}

func (s *service) runMACsQuery(ctx context.Context, q string) (map[string]string, error) {
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

func (s *service) queryMeasurements(ctx context.Context, columns []string, mac string, count int) ([]Fields, error) {
	columsParam := strings.Join(columns, ",")
	q := fmt.Sprintf(`SELECT %s
		FROM %s
		WHERE mac = $1
		ORDER BY time DESC
		LIMIT %d`, columsParam, s.cfg.Table, count)
	s.logger.LogAttrs(ctx, slog.LevelDebug, "RuuviTag values query", slog.String("mac", mac), slog.String("query", q))
	rows, err := s.dbpool.Query(ctx, q, mac)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var measurements []Fields
	for rows.Next() {
		f, err := Collect(rows, columns)
		if err != nil {
			return nil, err
		}
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Returned RuuviTag measurement", slog.Any("measurement", f))
		measurements = append(measurements, f)
	}
	return measurements, rows.Err()
}

func validColumns(columns []string) ([]string, error) {
	dedup := make(map[string]struct{})
	var valid []string
	for _, c := range columns {
		_, ok := queryColumnMap[c]
		if !ok {
			return nil, ErrInvalidColumn
		}
		_, seen := dedup[c]
		if !seen {
			valid = append(valid, c)
			dedup[c] = struct{}{}
		}
	}
	return valid, nil
}
