package measurement

import (
	"context"
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

const queryTemplate = `from(bucket: "%s")
  |> range(start: -1h)
  |> filter(fn: (r) =>
      r._measurement == "%s"
  )
  |> top(n:1, columns: ["name"])`

// Config is the InfluxDB connection config
type Config struct {
	Addr        string
	Username    string
	Password    string
	Database    string
	Measurement string
	Timeout     time.Duration
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Closer interface {
	Close() error
}

type Service interface {
	Pinger
	Closer
	Current(ctx context.Context) (map[string]Measurement, error)
}

type service struct {
	client   influxdb.Client
	queryAPI api.QueryAPI
	cfg      Config
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	token := fmt.Sprintf("%s:%s", cfg.Username, cfg.Password)
	client := influxdb.NewClient(cfg.Addr, token)
	return &service{
		client:   client,
		queryAPI: client.QueryAPI("temperature-api"),
		cfg:      cfg,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context) (map[string]Measurement, error) {
	q := fmt.Sprintf(queryTemplate, s.cfg.Database, s.cfg.Measurement)
	res, err := s.queryAPI.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	measurements := make(map[string]Measurement)
	for res.Next() {
		r := res.Record()
		name, ok := r.ValueByKey("name").(string)
		if !ok {
			continue
		}
		field, ok := r.ValueByKey("_field").(string)
		if !ok {
			continue
		}
		v, _ := r.ValueByKey("_value").(float64)
		m := measurements[name]
		m.Timestamp = r.Time()
		switch field {
		case "temperature":
			m.Temperature = v
		case "humidity":
			m.Humidity = v
		case "pressure":
			m.Pressure = v
		}
		measurements[name] = m
	}
	return measurements, res.Err()
}

// Ping checks that the server connection works
func (s *service) Ping(ctx context.Context) error {
	h, err := s.client.Health(ctx)
	if err != nil {
		return err
	}
	if h.Status != domain.HealthCheckStatusPass {
		return fmt.Errorf("%s", *h.Message)
	}
	return nil
}

func (s *service) Close() error {
	s.client.Close()
	return nil
}
