package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"

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
	if w.Code != http.StatusOK {
		t.Errorf("Wrong status code: %d", w.Code)
	}
	dec := json.NewDecoder(w.Body)
	m := make(map[string]interface{})
	if err := dec.Decode(&m); err != nil {
		t.Error(err)
	}
	lr := m["Living room"].(map[string]interface{})
	if lr["ts"].(string) != "2020-12-10T12:10:39Z" {
		t.Errorf("Invalid timestamp: %s", m["ts"])
	}
	if lr["temperature"].(float64) != 23.5 {
		t.Errorf("Invalid temperature: %f", m["temperature"])
	}
	if lr["humidity"].(float64) != 60.0 {
		t.Errorf("Invalid humidity: %f", m["humidity"])
	}
	if lr["pressure"].(float64) != 998.0 {
		t.Errorf("Invalid pressure: %f", m["pressure"])
	}
}
