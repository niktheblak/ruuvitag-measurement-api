package psql

import (
	"fmt"
	"strings"
	"time"
)

type Data struct {
	Addr              *string   `json:"mac,omitempty"`
	Name              *string   `json:"name,omitempty"`
	Temperature       *float64  `json:"temperature,omitempty"`
	Humidity          *float64  `json:"humidity,omitempty"`
	DewPoint          *float64  `json:"dew_point,omitempty"`
	Pressure          *float64  `json:"pressure,omitempty"`
	BatteryVoltage    *float64  `json:"battery_voltage,omitempty"`
	TxPower           *int      `json:"tx_power,omitempty"`
	AccelerationX     *int      `json:"acceleration_x,omitempty"`
	AccelerationY     *int      `json:"acceleration_y,omitempty"`
	AccelerationZ     *int      `json:"acceleration_z,omitempty"`
	MovementCounter   *int      `json:"movement_counter,omitempty"`
	MeasurementNumber *int      `json:"measurement_number,omitempty"`
	Timestamp         time.Time `json:"ts"`
}

type Scanner interface {
	Scan(dest ...any) error
}

func BuildQuery(table, nameTable string, columns []string) string {
	builder := new(strings.Builder)
	builder.WriteString("SELECT")
	var columnSelects []string
	for _, column := range columns {
		columnSelects = append(columnSelects, fmt.Sprintf(" %[1]s.%[2]s as \"%[2]s\"", table, column))
	}
	builder.WriteString(strings.Join(columnSelects, ","))
	builder.WriteString(fmt.Sprintf(" FROM %s", table))
	builder.WriteString(fmt.Sprintf(" JOIN (SELECT name, max(time) maxTime FROM %s GROUP BY name) b", table))
	builder.WriteString(fmt.Sprintf(" ON %[1]s.name = b.name AND %[1]s.time = b.maxTime", table))
	builder.WriteString(fmt.Sprintf(" WHERE %s.name IN (SELECT name FROM %s)", table, nameTable))
	return builder.String()
}

func Collect(res Scanner, columns []string) (Data, error) {
	// XXX: the *sql.Row.Scan(any...) function is a bit painful to work with
	// dynamic / configurable columns so this implementation is pretty gnarly. Beware!
	var d Data
	pointers := make([]any, len(columns))
	for i, column := range columns {
		switch column {
		case "time":
			pointers[i] = &d.Timestamp
		case "mac":
			var v string
			d.Addr = &v
			pointers[i] = d.Addr
		case "name":
			var v string
			d.Name = &v
			pointers[i] = d.Name
		case "temperature":
			var v float64
			d.Temperature = &v
			pointers[i] = d.Temperature
		case "humidity":
			var v float64
			d.Humidity = &v
			pointers[i] = d.Humidity
		case "pressure":
			var v float64
			d.Pressure = &v
			pointers[i] = d.Pressure
		case "battery_voltage":
			var v float64
			d.BatteryVoltage = &v
			pointers[i] = d.BatteryVoltage
		case "tx_power":
			var v int
			d.TxPower = &v
			pointers[i] = d.TxPower
		case "acceleration_x":
			var v int
			d.AccelerationX = &v
			pointers[i] = d.AccelerationX
		case "acceleration_y":
			var v int
			d.AccelerationY = &v
			pointers[i] = d.AccelerationY
		case "acceleration_z":
			var v int
			d.AccelerationZ = &v
			pointers[i] = d.AccelerationZ
		case "movement_counter":
			var v int
			d.MovementCounter = &v
			pointers[i] = d.MovementCounter
		case "measurement_number":
			var v int
			d.MeasurementNumber = &v
			pointers[i] = d.MeasurementNumber
		case "dew_point":
			var v float64
			d.DewPoint = &v
			pointers[i] = d.DewPoint
		default:
			return Data{}, fmt.Errorf("unknown column: %s", column)
		}
	}
	if err := res.Scan(pointers...); err != nil {
		return Data{}, err
	}
	return d, nil
}
