package measurement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

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
	Ping() error
}

type Service interface {
	Pinger
	Current(ctx context.Context) (map[string]Measurement, error)
}

type service struct {
	client   influxdb.Client
	database string
	query    string
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		Timeout:  cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("SELECT temperature, humidity, pressure FROM %s GROUP BY \"name\" ORDER BY \"time\" DESC LIMIT 1", cfg.Measurement)
	return &service{
		client:   client,
		database: cfg.Database,
		query:    q,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context) (map[string]Measurement, error) {
	q := influxdb.NewQuery(s.query, s.database, "")
	res, err := s.client.Query(q)
	if err != nil {
		return nil, err
	}
	m := make(map[string]Measurement)
	for _, r := range res.Results {
		for _, row := range r.Series {
			if len(row.Values) < 1 {
				continue
			}
			v := row.Values[0]
			if len(v) < 4 {
				continue
			}
			name, ok := row.Tags["name"]
			if !ok {
				continue
			}
			m[name] = Measurement{
				Timestamp:   parseTimestamp(v[0]),
				Temperature: parseFloat(v[1]),
				Humidity:    parseFloat(v[2]),
				Pressure:    parseFloat(v[3]),
			}
		}
	}
	return m, res.Error()
}

// Ping checks that the server connection works
func (s *service) Ping() error {
	_, _, err := s.client.Ping(5 * time.Second)
	return err
}

func parseTimestamp(v interface{}) time.Time {
	str, ok := v.(string)
	if !ok {
		return time.Time{}
	}
	ts, _ := time.Parse(time.RFC3339Nano, str)
	return ts
}

func parseFloat(v interface{}) float64 {
	n, ok := v.(json.Number)
	if ok {
		v, err := n.Float64()
		if err == nil {
			return v
		}
	}
	return 0
}
