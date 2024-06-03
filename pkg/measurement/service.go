package measurement

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/query"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
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

type Service interface {
	io.Closer
	Current(ctx context.Context) (map[string]sensor.Data, error)
}

type service struct {
	client   influxdb2.Client
	queryAPI api.QueryAPI
	cfg      Config
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	client := influxdb2.NewClientWithOptions(cfg.Addr, cfg.Token, influxdb2.DefaultOptions().
		SetUseGZip(true).
		SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
	return &service{
		client:   client,
		queryAPI: client.QueryAPI(cfg.Org),
		cfg:      cfg,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context) (measurements map[string]sensor.Data, err error) {
	q := fmt.Sprintf(queryTemplate, s.cfg.Bucket, s.cfg.Measurement)
	res, err := s.queryAPI.Query(ctx, q)
	if err != nil {
		return
	}
	defer func() {
		closeErr := res.Close()
		err = errors.Join(err, closeErr)
	}()
	measurements = make(map[string]sensor.Data)
	for res.Next() {
		r := res.Record()
		collate(r, measurements)
	}
	err = res.Err()
	return
}

func (s *service) Close() error {
	s.client.Close()
	return nil
}

func collate(r *query.FluxRecord, measurements map[string]sensor.Data) {
	name, ok := r.ValueByKey("name").(string)
	if !ok {
		return
	}
	field, ok := r.ValueByKey("_field").(string)
	if !ok {
		return
	}
	v, ok := r.ValueByKey("_value").(float64)
	if !ok {
		return
	}
	m := measurements[name]
	if m.Name == "" {
		m.Name = name
	}
	mac, ok := r.ValueByKey("mac").(string)
	if ok && m.Addr == "" {
		m.Addr = mac
	}
	if m.Timestamp.IsZero() {
		m.Timestamp = r.Time()
	}
	switch field {
	case "temperature":
		if m.Temperature == 0 {
			m.Temperature = v
		}
	case "humidity":
		if m.Humidity == 0 {
			m.Humidity = v
		}
	case "pressure":
		if m.Pressure == 0 {
			m.Pressure = v
		}
	case "dew_point":
		if m.DewPoint == 0 {
			m.DewPoint = v
		}
	case "battery_voltage":
		if m.BatteryVoltage == 0 {
			m.BatteryVoltage = v
		}
	case "tx_power":
		if m.TxPower == 0 {
			m.TxPower = int(v)
		}
	case "acceleration_x":
		if m.AccelerationX == 0 {
			m.AccelerationX = int(v)
		}
	case "acceleration_y":
		if m.AccelerationY == 0 {
			m.AccelerationY = int(v)
		}
	case "acceleration_z":
		if m.AccelerationZ == 0 {
			m.AccelerationZ = int(v)
		}
	case "movement_counter":
		if m.MovementCounter == 0 {
			m.MovementCounter = int(v)
		}
	case "measurement_number":
		if m.MeasurementNumber == 0 {
			m.MeasurementNumber = int(v)
		}
	}
	measurements[name] = m
}
