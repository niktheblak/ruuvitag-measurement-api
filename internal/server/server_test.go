package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niktheblak/web-common/pkg/auth"
)

const testAccessToken = "a65cd12f9bba453"

type mockService struct {
	Response []sensor.Fields
	Location *time.Location
}

func (s *mockService) Current(ctx context.Context, columns []string) (measurements map[string]sensor.Fields, err error) {
	if s.Response != nil {
		response := make(map[string]sensor.Fields)
		for _, v := range s.Response {
			v.Timestamp = v.Timestamp.In(s.Location)
			response[*v.Name] = v
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
	srv := New(svc, sensor.DefaultColumnMap, auth.Static(testAccessToken), nil)
	t.Run("with token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", testAccessToken))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		m := decode(t, w.Body)
		require.Contains(t, m, "Living Room")
		lr := m["Living Room"]
		assert.Equal(t, "2020-12-10T12:10:39Z", lr["time"])
		assert.Equal(t, 23.5, lr["temperature"])
		assert.Equal(t, 60.0, lr["humidity"])
		assert.Equal(t, 998.0, lr["pressure"])
		assert.Equal(t, 102.0, lr["movement_counter"])
		assert.Equal(t, 71.0, lr["measurement_number"])
	})
	t.Run("without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
	t.Run("timezone", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?tz=Europe/Helsinki", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", testAccessToken))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		res := decode(t, w.Body)
		require.Contains(t, res, "Living Room")
		lr := res["Living Room"]
		assert.Equal(t, "2020-12-10T14:10:39+02:00", lr["time"])
	})
}

func decode(t *testing.T, r io.Reader) map[string]map[string]any {
	dec := json.NewDecoder(r)
	results := make(map[string]map[string]any)
	if err := dec.Decode(&results); err != nil {
		t.Fatal(err)
	}
	return results
}
