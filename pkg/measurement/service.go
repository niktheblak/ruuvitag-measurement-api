package measurement

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const queryTemplate = `from(bucket: "%s")
  |> range(start: -1h)
  |> filter(fn: (r) =>
      r._measurement == "%s"
  )
  |> top(n:1, columns: ["_time", "name"])`

// Config is the InfluxDB connection config
type Config struct {
	Addr        string
	Org         string
	Token       string
	Bucket      string
	Measurement string
	Timeout     time.Duration
}

type Closer interface {
	Close() error
}

type Service interface {
	Closer
	Current(ctx context.Context) (map[string]Measurement, error)
}

type service struct {
	client   influxdb2.Client
	queryAPI api.QueryAPI
	cfg      Config
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	client := influxdb2.NewClient(cfg.Addr, cfg.Token)
	return &service{
		client:   client,
		queryAPI: client.QueryAPI(cfg.Org),
		cfg:      cfg,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context) (map[string]Measurement, error) {
	q := fmt.Sprintf(queryTemplate, s.cfg.Bucket, s.cfg.Measurement)
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
		case "dew_point":
			m.DewPoint = v
		}
		measurements[name] = m
	}
	return measurements, res.Err()
}

func (s *service) Close() error {
	s.client.Close()
	return nil
}
