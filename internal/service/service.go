package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

type Measurement struct {
	Timestamp   time.Time
	Temperature float64
	Humidity    float64
	Pressure    float64
}

type Service struct {
	client   influxdb.Client
	database string
	query    string
}

func New(addr, username, password, database, measurement string) (*Service, error) {
	cfg := influxdb.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	}
	client, err := influxdb.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("SELECT temperature, humidity, pressure FROM %s GROUP BY \"name\" LIMIT 1", measurement)
	return &Service{
		client:   client,
		database: database,
		query:    q,
	}, nil
}

func (s *Service) Current(ctx context.Context) (map[string]Measurement, error) {
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

func (s *Service) Ping() error {
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
