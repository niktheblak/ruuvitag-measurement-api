package sensor

import (
	"time"
)

type Data struct {
	Addr              string    `json:"mac"`
	Name              string    `json:"name,omitempty"`
	Temperature       float64   `json:"temperature"`
	Humidity          float64   `json:"humidity"`
	DewPoint          float64   `json:"dew_point,omitempty"`
	Pressure          float64   `json:"pressure"`
	BatteryVoltage    float64   `json:"battery_voltage,omitempty"`
	TxPower           int       `json:"tx_power,omitempty"`
	AccelerationX     int       `json:"acceleration_x,omitempty"`
	AccelerationY     int       `json:"acceleration_y,omitempty"`
	AccelerationZ     int       `json:"acceleration_z,omitempty"`
	MovementCounter   int       `json:"movement_counter,omitempty"`
	MeasurementNumber int       `json:"measurement_number,omitempty"`
	Timestamp         time.Time `json:"ts"`
}
