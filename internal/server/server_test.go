package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niktheblak/temperature-api/pkg/measurement"
)

type mockService struct {
}

func (s *mockService) Current(ctx context.Context) (map[string]measurement.Measurement, error) {
	return map[string]measurement.Measurement{
		"Living room": {
			Timestamp:   time.Date(2020, time.December, 10, 12, 10, 39, 0, time.UTC),
			Temperature: 23.5,
			Humidity:    60.0,
			Pressure:    998.0,
		},
	}, nil
}

func (s *mockService) Ping() error {
	return nil
}

func (s *mockService) Close() error {
	return nil
}

func TestServe(t *testing.T) {
	srv := &Server{
		Service: &mockService{},
		Router:  httprouter.New(),
	}
	srv.Routes()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	dec := json.NewDecoder(w.Body)
	m := make(map[string]interface{})
	err := dec.Decode(&m)
	require.NoError(t, err)
	require.IsType(t, map[string]interface{}{}, m["Living room"])
	lr := m["Living room"].(map[string]interface{})
	assert.Equal(t, "2020-12-10T12:10:39Z", lr["ts"])
	assert.Equal(t, 23.5, lr["temperature"])
	assert.Equal(t, 60.0, lr["humidity"])
	assert.Equal(t, 998.0, lr["pressure"])
}
