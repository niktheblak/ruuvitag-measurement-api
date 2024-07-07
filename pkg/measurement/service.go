package measurement

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"time"

	_ "github.com/lib/pq"

	"github.com/niktheblak/temperature-api/pkg/psql"
)

type Config struct {
	PsqlInfo  string
	Table     string
	NameTable string
	Columns   map[string]string
	Logger    *slog.Logger
}

type Service interface {
	Current(ctx context.Context, loc *time.Location) (measurements map[string]psql.Data, err error)
	io.Closer
}

type service struct {
	db      *sql.DB
	columns []string
	q       string
	logger  *slog.Logger
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
	db, err := sql.Open("postgres", cfg.PsqlInfo)
	if err != nil {
		return nil, err
	}
	var columnList []string
	for _, cn := range cfg.Columns {
		columnList = append(columnList, cn)
	}
	sort.Strings(columnList)
	q := psql.BuildQuery(cfg.Table, cfg.NameTable, columnList)
	cfg.Logger.LogAttrs(nil, slog.LevelDebug, "Rendered query", slog.String("query", q))
	return &service{
		db:      db,
		columns: columnList,
		q:       q,
		logger:  cfg.Logger,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context, loc *time.Location) (measurements map[string]psql.Data, err error) {
	res, err := s.db.QueryContext(ctx, s.q)
	if err != nil {
		return
	}
	measurements = make(map[string]psql.Data)
	for res.Next() {
		d, err := psql.Collect(res, s.columns)
		if err != nil {
			return nil, err
		}
		d.Timestamp = d.Timestamp.In(loc)
		var name string
		if d.Name != nil {
			name = *d.Name
		} else if d.Addr != nil {
			name = *d.Addr
		} else {
			return nil, fmt.Errorf("column name or address is required")
		}
		measurements[name] = d
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Found measurement", slog.Any("data", d))
	}
	err = res.Err()
	return
}

func (s *service) Close() error {
	return s.db.Close()
}
