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
	Timestamp         time.Time `json:"time"`
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
			d.Addr = ZeroStringPointer()
			pointers[i] = d.Addr
		case "name":
			d.Name = ZeroStringPointer()
			pointers[i] = d.Name
		case "temperature":
			d.Temperature = ZeroFloat64Pointer()
			pointers[i] = d.Temperature
		case "humidity":
			d.Humidity = ZeroFloat64Pointer()
			pointers[i] = d.Humidity
		case "pressure":
			d.Pressure = ZeroFloat64Pointer()
			pointers[i] = d.Pressure
		case "battery_voltage":
			d.BatteryVoltage = ZeroFloat64Pointer()
			pointers[i] = d.BatteryVoltage
		case "tx_power":
			d.TxPower = ZeroIntPointer()
			pointers[i] = d.TxPower
		case "acceleration_x":
			d.AccelerationX = ZeroIntPointer()
			pointers[i] = d.AccelerationX
		case "acceleration_y":
			d.AccelerationY = ZeroIntPointer()
			pointers[i] = d.AccelerationY
		case "acceleration_z":
			d.AccelerationZ = ZeroIntPointer()
			pointers[i] = d.AccelerationZ
		case "movement_counter":
			d.MovementCounter = ZeroIntPointer()
			pointers[i] = d.MovementCounter
		case "measurement_number":
			d.MeasurementNumber = ZeroIntPointer()
			pointers[i] = d.MeasurementNumber
		case "dew_point":
			d.DewPoint = ZeroFloat64Pointer()
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

func Float64Pointer(v float64) *float64 {
	return &v
}

func ZeroFloat64Pointer() *float64 {
	var v float64
	return &v
}

func IntPointer(v int) *int {
	return &v
}

func ZeroIntPointer() *int {
	var v int
	return &v
}

func StringPointer(v string) *string {
	return &v
}

func ZeroStringPointer() *string {
	var v string
	return &v
}
