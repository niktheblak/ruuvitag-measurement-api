package ruuvitag

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
)

var (
	namesTmpl = template.Must(template.New("SelectNames").Parse(`
		SELECT {{.Name}} FROM {{.Table}} ORDER BY {{.Name}}
	`))
	measurementsTmpl = template.Must(template.New("SelectMeasurements").Parse(`
		SELECT {{.Columns}}
		FROM {{.Table}}
		WHERE {{.NameColumn}} = $1
		ORDER BY {{.TimeColumn}} DESC
		LIMIT {{.Limit}}
	`))
)

type namesTmplValues struct {
	Name  string
	Table string
}

type measurementsTmplValues struct {
	Columns    string
	Table      string
	NameColumn string
	TimeColumn string
	Limit      int
}

func init() {

}

type Scanner interface {
	Scan(dest ...any) error
}

type QueryBuilder struct {
	Table     string
	NameTable string
	Columns   map[string]string
}

func (q *QueryBuilder) Names() (string, error) {
	b := new(strings.Builder)
	err := namesTmpl.Execute(b, namesTmplValues{
		Name:  q.Columns["name"],
		Table: q.NameTable,
	})
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (q *QueryBuilder) Latest(columns []string, n int) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("no columns specified")
	}
	if n < 1 {
		return "", fmt.Errorf("n must be at least 1")
	}
	b := new(strings.Builder)
	err := measurementsTmpl.Execute(b, measurementsTmplValues{
		Columns:    strings.Join(columns, ","),
		Table:      q.Table,
		NameColumn: q.Columns["name"],
		TimeColumn: q.Columns["time"],
		Limit:      n,
	})
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (q *QueryBuilder) Collect(res Scanner, columns []string) (sensor.Fields, error) {
	// XXX: the *sql.Row.Scan(any...) function is a bit painful to work with
	// dynamic / configurable columns so this implementation is pretty gnarly. Beware!
	var d sensor.Fields
	pointers := make([]any, len(columns))
	for i, column := range columns {
		switch column {
		case q.Columns["time"]:
			pointers[i] = &d.Timestamp
		case q.Columns["mac"]:
			d.Addr = sensor.ZeroStringPointer()
			pointers[i] = d.Addr
		case q.Columns["name"]:
			d.Name = sensor.ZeroStringPointer()
			pointers[i] = d.Name
		case q.Columns["temperature"]:
			d.Temperature = sensor.ZeroFloat64Pointer()
			pointers[i] = d.Temperature
		case q.Columns["humidity"]:
			d.Humidity = sensor.ZeroFloat64Pointer()
			pointers[i] = d.Humidity
		case q.Columns["pressure"]:
			d.Pressure = sensor.ZeroFloat64Pointer()
			pointers[i] = d.Pressure
		case q.Columns["battery_voltage"]:
			d.BatteryVoltage = sensor.ZeroFloat64Pointer()
			pointers[i] = d.BatteryVoltage
		case q.Columns["tx_power"]:
			d.TxPower = sensor.ZeroIntPointer()
			pointers[i] = d.TxPower
		case q.Columns["acceleration_x"]:
			d.AccelerationX = sensor.ZeroIntPointer()
			pointers[i] = d.AccelerationX
		case q.Columns["acceleration_y"]:
			d.AccelerationY = sensor.ZeroIntPointer()
			pointers[i] = d.AccelerationY
		case q.Columns["acceleration_z"]:
			d.AccelerationZ = sensor.ZeroIntPointer()
			pointers[i] = d.AccelerationZ
		case q.Columns["movement_counter"]:
			d.MovementCounter = sensor.ZeroIntPointer()
			pointers[i] = d.MovementCounter
		case q.Columns["measurement_number"]:
			d.MeasurementNumber = sensor.ZeroIntPointer()
			pointers[i] = d.MeasurementNumber
		case q.Columns["dew_point"]:
			d.DewPoint = sensor.ZeroFloat64Pointer()
			pointers[i] = d.DewPoint
		default:
			return d, fmt.Errorf("unknown column: %s", column)
		}
	}
	if err := res.Scan(pointers...); err != nil {
		return d, err
	}
	return d, nil
}

func CleanForLogging(query string) string {
	r := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(r.ReplaceAllString(query, " "))
}
