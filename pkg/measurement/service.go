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
	"github.com/spf13/cast"
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
	rawValue := r.ValueByKey("_value")
	if rawValue == nil {
		return
	}
	v := cast.ToFloat64(rawValue)
	m := measurements[name]
	if m.Name == "" {
		m.Name = name
	}
	mac, _ := r.ValueByKey("mac").(string)
	m.Addr = mac
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
	case "battery_voltage":
		m.BatteryVoltage = v
	case "tx_power":
		m.TxPower = int(v)
	case "acceleration_x":
		m.AccelerationX = int(v)
	case "acceleration_y":
		m.AccelerationY = int(v)
	case "acceleration_z":
		m.AccelerationZ = int(v)
	case "movement_counter":
		m.MovementCounter = int(v)
	case "measurement_number":
		m.MeasurementNumber = int(v)
	}
	measurements[name] = m
}
