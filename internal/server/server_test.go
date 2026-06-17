package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	Response []sensor.Fields
	Location *time.Location
}

func (s *mockService) Latest(ctx context.Context, n int, columns []string, names []string) (measurements map[string][]sensor.Fields, err error) {
	if s.Response != nil {
		response := make(map[string][]sensor.Fields)
		for _, v := range s.Response {
			v.Timestamp = v.Timestamp.In(s.Location)
			response[*v.Name] = []sensor.Fields{v}
		}
		return response, nil
	}
	return nil, nil
}

func (s *mockService) Ping(ctx context.Context) error {
	return nil
}

func (s *mockService) Close() error {
	return nil
}

func TestServe(t *testing.T) {
	svc := &mockService{
		Response: []sensor.Fields{
			{
				Addr:              sensor.StringPointer("31:1a:a3:af:72:93"),
				Timestamp:         time.Date(2020, time.December, 10, 12, 10, 39, 0, time.UTC),
				Name:              sensor.StringPointer("Living Room"),
				Temperature:       sensor.Float64Pointer(23.5),
				Humidity:          sensor.Float64Pointer(60.0),
				Pressure:          sensor.Float64Pointer(998.0),
				BatteryVoltage:    sensor.Float64Pointer(1.75),
				TxPower:           sensor.IntPointer(11),
				MovementCounter:   sensor.IntPointer(102),
				MeasurementNumber: sensor.IntPointer(71),
			},
		},
		Location: time.UTC,
	}
	srv := New(svc, sensor.DefaultColumnMap, nil)
	t.Run("get latest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		res := decode(t, w.Body)
		lr := res.Series[0].Measurements[0]
		assert.Equal(t, "2020-12-10T12:10:39Z", lr["time"])
		assert.Equal(t, 23.5, lr["temperature"])
		assert.Equal(t, 60.0, lr["humidity"])
		assert.Equal(t, 998.0, lr["pressure"])
		assert.Equal(t, 102.0, lr["movement_counter"])
		assert.Equal(t, 71.0, lr["measurement_number"])
	})
	t.Run("timezone", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?tz=Europe/Helsinki", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		res := decode(t, w.Body)
		lr := res.Series[0].Measurements[0]
		assert.Equal(t, "2020-12-10T14:10:39+02:00", lr["time"])
	})
}

func decode(t *testing.T, r io.Reader) response {
	dec := json.NewDecoder(r)
	var res response
	if err := dec.Decode(&res); err != nil {
		t.Fatal(err)
	}
	return res
}
