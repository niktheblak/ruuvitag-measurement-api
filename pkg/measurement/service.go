package measurement

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"text/template"
	"time"

	_ "github.com/lib/pq"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
)

const queryTemplate = `SELECT
	{{.Table}}.{{.TimeColumn}} as "{{.TimeColumn}}",
	{{.Table}}.mac as "mac",
	{{.Table}}.name as "name",
	{{.Table}}.temperature as "temperature",
	{{.Table}}.humidity as "humidity",
	{{.Table}}.pressure as "pressure",
	{{.Table}}.battery_voltage as "battery_voltage",
	{{.Table}}.tx_power as "tx_power",
	{{.Table}}.acceleration_x as "acceleration_x",
	{{.Table}}.acceleration_y as "acceleration_y",
	{{.Table}}.acceleration_z as "acceleration_z",
	{{.Table}}.movement_counter as "movement_counter",
	{{.Table}}.measurement_number as "measurement_number",
	{{.Table}}.dew_point as "dew_point"
FROM {{.Table}}
JOIN (SELECT name, max(time) maxTime
	FROM {{.Table}}
	GROUP BY name) b
ON {{.Table}}.name = b.name AND {{.Table}}.time = b.maxTime
WHERE {{.Table}}.name IN (SELECT name FROM {{.NameTable}});`

type Config struct {
	PsqlInfo   string
	Table      string
	NameTable  string
	TimeColumn string
	Logger     *slog.Logger
}

type Service interface {
	Current(ctx context.Context) (measurements map[string]sensor.Data, err error)
	io.Closer
}

type service struct {
	db     *sql.DB
	cfg    Config
	q      string
	logger *slog.Logger
}

// New creates a new instance of the service using the given config
func New(cfg Config) (Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	db, err := sql.Open("postgres", cfg.PsqlInfo)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("query").Parse(queryTemplate)
	if err != nil {
		return nil, err
	}
	builder := new(strings.Builder)
	if err := tmpl.Execute(builder, cfg); err != nil {
		return nil, err
	}
	cfg.Logger.LogAttrs(nil, slog.LevelDebug, "Rendered query", slog.String("query", builder.String()))
	return &service{
		db:     db,
		cfg:    cfg,
		q:      builder.String(),
		logger: cfg.Logger,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context) (measurements map[string]sensor.Data, err error) {
	res, err := s.db.QueryContext(ctx, s.q)
	if err != nil {
		return
	}
	measurements = make(map[string]sensor.Data)
	for res.Next() {
		var (
			timestamp         time.Time
			addr              string
			name              string
			temperature       float64
			humidity          float64
			pressure          float64
			batteryVoltage    *float64
			txPower           *int
			accelerationX     *int
			accelerationY     *int
			accelerationZ     *int
			movementCounter   *int
			measurementNumber *int
			dewPoint          *float64
		)
		err = res.Scan(
			&timestamp,
			&addr,
			&name,
			&temperature,
			&humidity,
			&pressure,
			&batteryVoltage,
			&txPower,
			&accelerationX,
			&accelerationY,
			&accelerationZ,
			&movementCounter,
			&measurementNumber,
			&dewPoint,
		)
		if err != nil {
			return
		}
		if batteryVoltage == nil {
			var z float64
			batteryVoltage = &z
		}
		if txPower == nil {
			var z int
			txPower = &z
		}
		if accelerationX == nil {
			var z int
			accelerationX = &z
		}
		if accelerationY == nil {
			var z int
			accelerationY = &z
		}
		if accelerationZ == nil {
			var z int
			accelerationZ = &z
		}
		if movementCounter == nil {
			var z int
			movementCounter = &z
		}
		if measurementNumber == nil {
			var z int
			measurementNumber = &z
		}
		if dewPoint == nil {
			var z float64
			dewPoint = &z
		}
		measurements[name] = sensor.Data{
			Timestamp:         timestamp,
			Addr:              addr,
			Name:              name,
			Temperature:       temperature,
			Humidity:          humidity,
			Pressure:          pressure,
			BatteryVoltage:    *batteryVoltage,
			TxPower:           *txPower,
			AccelerationX:     *accelerationX,
			AccelerationY:     *accelerationY,
			AccelerationZ:     *accelerationZ,
			MovementCounter:   *movementCounter,
			MeasurementNumber: *measurementNumber,
			DewPoint:          *dewPoint,
		}
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Found measurements", slog.Any("measurements", measurements))
	}
	err = res.Err()
	return
}

func (s *service) Close() error {
	return s.db.Close()
}
