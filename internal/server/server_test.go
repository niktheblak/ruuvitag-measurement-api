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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"

	"github.com/niktheblak/temperature-api/pkg/auth"
)

const testAccessToken = "a65cd12f9bba453"

type mockService struct {
	Response map[string]sensor.Data
}

func (s *mockService) Current(ctx context.Context) (map[string]sensor.Data, error) {
	if s.Response != nil {
		return s.Response, nil
	}
	return map[string]sensor.Data{}, nil
}

func (s *mockService) Close() error {
	return nil
}

func TestServe(t *testing.T) {
	svc := new(mockService)
	svc.Response = map[string]sensor.Data{
		"Living room": {
			Timestamp:         time.Date(2020, time.December, 10, 12, 10, 39, 0, time.UTC),
			Temperature:       23.5,
			Humidity:          60.0,
			Pressure:          998.0,
			BatteryVoltage:    1.75,
			TxPower:           11,
			MovementCounter:   102,
			MeasurementNumber: 71,
		},
	}
	srv := New(svc, auth.Static(testAccessToken), nil)
	t.Run("with token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", testAccessToken))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		m := decode(t, w.Body)
		require.IsType(t, map[string]interface{}{}, m["Living room"])
		lr := m["Living room"].(map[string]interface{})
		assert.Equal(t, "2020-12-10T12:10:39Z", lr["ts"])
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
		m := decode(t, w.Body)
		require.IsType(t, map[string]interface{}{}, m["Living room"])
		lr := m["Living room"].(map[string]interface{})
		assert.Equal(t, "2020-12-10T14:10:39+02:00", lr["ts"])
	})
}

func decode(t *testing.T, r io.Reader) map[string]interface{} {
	dec := json.NewDecoder(r)
	m := make(map[string]interface{})
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}
