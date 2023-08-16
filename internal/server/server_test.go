package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niktheblak/temperature-api/pkg/auth"
	"github.com/niktheblak/temperature-api/pkg/measurement"
)

const testAccessToken = "a65cd12f9bba453"

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
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(svc, auth.Static(testAccessToken), logger)
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
