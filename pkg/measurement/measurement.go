package measurement

import "time"

// Measurement is one RuuviTag measurement
type Measurement struct {
	Timestamp   time.Time `json:"ts"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Pressure    float64   `json:"pressure"`
}
