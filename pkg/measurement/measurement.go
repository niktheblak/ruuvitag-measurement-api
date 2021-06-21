package measurement

import "time"

// Measurement is one RuuviTag measurement
type Measurement struct {
	Timestamp   time.Time
	Temperature float64
	Humidity    float64
	Pressure    float64
	DewPoint    float64
}
