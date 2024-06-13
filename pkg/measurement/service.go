package measurement

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
)

const queryTemplate = `SELECT
		%[1]s.%[2]s as "%[2]s",
		%[1]s.mac as "mac",
		%[1]s.name as "name",
		%[1]s.temperature as "temperature",
		%[1]s.humidity as "humidity",
		%[1]s.pressure as "pressure",
		%[1]s.battery_voltage as "battery_voltage",
		%[1]s.tx_power as "tx_power",
		%[1]s.acceleration_x as "acceleration_x",
		%[1]s.acceleration_y as "acceleration_y",
		%[1]s.acceleration_z as "acceleration_z",
		%[1]s.movement_counter as "movement_counter",
		%[1]s.measurement_number as "measurement_number",
		%[1]s.dew_point as "dew_point"
	FROM %[1]s
	JOIN (SELECT name, max(time) maxTime
	  FROM %[1]s
	  GROUP BY name) b
	ON %[1]s.name = b.name AND %[1]s.time = b.maxTime
	WHERE %[1]s.name IN (SELECT name FROM %[3]s);`

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
	return &service{
		db:     db,
		cfg:    cfg,
		logger: cfg.Logger,
	}, nil
}

// Current returns current measurements
func (s *service) Current(ctx context.Context) (measurements map[string]sensor.Data, err error) {
	q := fmt.Sprintf(queryTemplate, s.cfg.Table, s.cfg.TimeColumn, s.cfg.NameTable)
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Querying current measurements", slog.String("query", q))
	res, err := s.db.QueryContext(ctx, q)
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
