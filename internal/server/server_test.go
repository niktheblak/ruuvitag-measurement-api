package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niktheblak/ruuvitag-measurement-api/pkg/ruuvitag"
)

type mockService struct {
	Response []ruuvitag.Fields
	Location *time.Location
}

func (s *mockService) Latest(ctx context.Context, columns []string, names []string, count int) (measurements map[string][]ruuvitag.Fields, err error) {
	if s.Response != nil {
		resp := make(map[string][]ruuvitag.Fields)
		for _, v := range s.Response {
			v.Timestamp = v.Timestamp.In(s.Location)
			resp[*v.Name] = []ruuvitag.Fields{v}
		}
		return resp, nil
	}
	return nil, nil
}

func (s *mockService) Ping(ctx context.Context) error {
	return nil
}

func (s *mockService) Close() error {
	return nil
}

type testResponse struct {
	Timezone string           `json:"tz,omitempty"`
	Columns  []string         `json:"columns"`
	Series   []testSeriesItem `json:"series"`
}

type testSeriesItem struct {
	Name         string           `json:"name,omitempty"`
	MAC          string           `json:"mac"`
	Measurements []map[string]any `json:"measurements"`
}

func TestServe(t *testing.T) {
	svc := &mockService{
		Response: []ruuvitag.Fields{
			{
				Addr:              new("31:1a:a3:af:72:93"),
				Timestamp:         time.Date(2020, time.December, 10, 12, 10, 39, 0, time.UTC),
				Name:              new("Living Room"),
				Temperature:       new(23.5),
				Humidity:          new(60.0),
				Pressure:          new(998.0),
				BatteryVoltage:    new(1.75),
				MovementCounter:   new(102),
				MeasurementNumber: new(71),
			},
		},
		Location: time.UTC,
	}
	srv := New(svc, nil)
	t.Run("get latest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		res := decode(t, w.Body)
		m := res.Series[0].Measurements[0]
		assert.Equal(t, "2020-12-10T12:10:39Z", m["time"])
		assert.Equal(t, 23.5, m["temperature"])
		assert.Equal(t, 60.0, m["humidity"])
		assert.Equal(t, 998.0, m["pressure"])
		assert.Equal(t, 102.0, m["movement_counter"])
		assert.Equal(t, 71.0, m["measurement_number"])
	})
	t.Run("timezone", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?tz=Europe/Helsinki", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		res := decode(t, w.Body)
		m := res.Series[0].Measurements[0]
		assert.Equal(t, "2020-12-10T14:10:39+02:00", m["time"])
	})
}

func decode(t *testing.T, r io.Reader) testResponse {
	dec := json.NewDecoder(r)
	var res testResponse
	if err := dec.Decode(&res); err != nil {
		t.Fatal(err)
	}
	return res
}
