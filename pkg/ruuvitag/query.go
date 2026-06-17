package ruuvitag

import (
	"encoding/json"
	"fmt"
	"time"
)

type Scanner interface {
	Scan(dest ...any) error
}

type Fields struct {
	Timestamp         time.Time `json:"time"`
	Addr              *string   `json:"mac,omitempty"`
	Name              *string   `json:"name,omitempty"`
	Temperature       *float64  `json:"temperature,omitempty"`
	Humidity          *float64  `json:"humidity,omitempty"`
	DewPoint          *float64  `json:"dew_point,omitempty"`
	WetBulb           *float64  `json:"wet_bulb,omitempty"`
	Pressure          *float64  `json:"pressure,omitempty"`
	BatteryVoltage    *float64  `json:"battery_voltage,omitempty"`
	AccelerationX     *int      `json:"acceleration_x,omitempty"`
	AccelerationY     *int      `json:"acceleration_y,omitempty"`
	AccelerationZ     *int      `json:"acceleration_z,omitempty"`
	MovementCounter   *int      `json:"movement_counter,omitempty"`
	MeasurementNumber *int      `json:"measurement_number,omitempty"`
}

func (f *Fields) String() string {
	js, err := json.Marshal(*f)
	if err != nil {
		panic(err)
	}
	return string(js)
}

func Collect(res Scanner, columns []string) (Fields, error) {
	var f Fields
	pointers := make([]any, len(columns))
	for i, column := range columns {
		switch column {
		case "time":
			pointers[i] = &f.Timestamp
		case "mac":
			pointers[i] = &f.Addr
		case "name":
			f.Name = new(string)
			pointers[i] = f.Name
		case "temperature":
			f.Temperature = new(float64)
			pointers[i] = f.Temperature
		case "humidity":
			f.Humidity = new(float64)
			pointers[i] = f.Humidity
		case "pressure":
			f.Pressure = new(float64)
			pointers[i] = f.Pressure
		case "battery_voltage":
			f.BatteryVoltage = new(float64)
			pointers[i] = f.BatteryVoltage
		case "acceleration_x":
			f.AccelerationX = new(int)
			pointers[i] = f.AccelerationX
		case "acceleration_y":
			f.AccelerationY = new(int)
			pointers[i] = f.AccelerationY
		case "acceleration_z":
			f.AccelerationZ = new(int)
			pointers[i] = f.AccelerationZ
		case "movement_counter":
			f.MovementCounter = new(int)
			pointers[i] = f.MovementCounter
		case "measurement_number":
			f.MeasurementNumber = new(int)
			pointers[i] = f.MeasurementNumber
		case "dew_point":
			f.DewPoint = new(float64)
			pointers[i] = f.DewPoint
		case "wet_bulb":
			f.WetBulb = new(float64)
			pointers[i] = f.WetBulb
		default:
			return f, fmt.Errorf("unknown column: %s", column)
		}
	}
	if err := res.Scan(pointers...); err != nil {
		return f, err
	}
	return f, nil
}
