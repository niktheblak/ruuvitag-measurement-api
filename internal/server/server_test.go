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

	"github.com/niktheblak/temperature-api/pkg/measurement"
)

type mockService struct {
	Response     map[string]measurement.Measurement
	PingResponse error
}

func (s *mockService) Current(ctx context.Context) (map[string]measurement.Measurement, error) {
	if s.Response != nil {
		return s.Response, nil
	}
	return map[string]measurement.Measurement{}, nil
}

func (s *mockService) Ping(ctx context.Context) error {
	return s.PingResponse
}

func (s *mockService) Close() error {
	return nil
}

func TestServe(t *testing.T) {
	svc := new(mockService)
	svc.Response = map[string]measurement.Measurement{
		"Living room": {
			Timestamp:   time.Date(2020, time.December, 10, 12, 10, 39, 0, time.UTC),
			Temperature: 23.5,
			Humidity:    60.0,
			Pressure:    998.0,
		},
	}
	srv := New(svc)
	req := httptest.NewRequest("GET", "/", nil)
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
}

func TestHealth(t *testing.T) {
	svc := new(mockService)
	srv := New(svc)
	t.Run("Health OK", func(t *testing.T) {
		svc.PingResponse = nil
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		m := decode(t, w.Body)
		assert.Equal(t, "ok", m["status"])
	})
	t.Run("Health error", func(t *testing.T) {
		svc.PingResponse = fmt.Errorf("database error")
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		m := decode(t, w.Body)
		assert.Equal(t, "error", m["status"])
		assert.Equal(t, "database error", m["error"])
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
