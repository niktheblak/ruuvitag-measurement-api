package ruuvitag

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
)

type Scanner interface {
	Scan(dest ...any) error
}

var (
	namesTmpl = template.Must(template.New("SelectNames").Parse(`
		SELECT {{.MACColumn}}, {{.NameColumn}}
        FROM {{.Table}}
        WHERE {{.NameColumn}} = ANY(ARRAY[{{.Names}}])
	`))
	measurementsTmpl = template.Must(template.New("SelectMeasurements").Parse(`
		SELECT {{.Columns}}
		FROM {{.Table}}
		WHERE {{.MACColumn}} = $1
		ORDER BY {{.TimeColumn}} DESC
		LIMIT {{.Limit}}
	`))
)

type QueryBuilder struct {
	Table      string
	NameTable  string
	NameColumn string
	Columns    map[string]string
}

func (q *QueryBuilder) Names(names []string) string {
	var quotedNames []string
	for _, name := range names {
		quotedNames = append(quotedNames, fmt.Sprintf(`'%s'`, name))
	}
	type namesTmplValues struct {
		Table      string
		MACColumn  string
		NameColumn string
		Names      string
	}
	var b strings.Builder
	err := namesTmpl.Execute(&b, namesTmplValues{
		Names:      strings.Join(quotedNames, ","),
		Table:      q.NameTable,
		MACColumn:  q.Columns["mac"],
		NameColumn: q.NameColumn,
	})
	if err != nil {
		panic(err)
	}
	return b.String()
}

func (q *QueryBuilder) Latest(columns []string, n int) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("no columns specified")
	}
	if n < 1 {
		return "", fmt.Errorf("n must be at least 1")
	}
	type measurementsTmplValues struct {
		Columns    string
		Table      string
		MACColumn  string
		TimeColumn string
		Limit      int
	}

	var b strings.Builder
	err := measurementsTmpl.Execute(&b, measurementsTmplValues{
		Columns:    strings.Join(columns, ","),
		Table:      q.Table,
		MACColumn:  q.Columns["mac"],
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
		case q.NameColumn:
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
		case q.Columns["wet_bulb"]:
			d.WetBulb = sensor.ZeroFloat64Pointer()
			pointers[i] = d.WetBulb
		default:
			return d, fmt.Errorf("unknown column: %s", column)
		}
	}
	if err := res.Scan(pointers...); err != nil {
		return d, err
	}
	return d, nil
}
