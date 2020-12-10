package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/niktheblak/temperature-api/pkg/measurement"
)

type jsonMeasurement struct {
	Timestamp   string  `json:"ts"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Pressure    float64 `json:"pressure"`
}

type Server struct {
	Service measurement.Service
}

func New(svc measurement.Service) *Server {
	return &Server{Service: svc}
}

func (s *Server) Current(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	measurements, err := s.Service.Current(r.Context())
	if err != nil {
		log.Printf("Error while reading response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	loc, err := location(r.URL.Query().Get("tz"))
	if err != nil {
		log.Printf("Invalid location: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("ETag", etag(measurements))
	js := toJSON(measurements, loc)
	if err := json.NewEncoder(w).Encode(js); err != nil {
		log.Fatal(err)
	}
}

func location(tz string) (loc *time.Location, err error) {
	if tz != "" {
		loc, err = time.LoadLocation(tz)
		return
	}
	loc = time.UTC
	return
}

func etag(measurements map[string]measurement.Measurement) string {
	if len(measurements) == 0 {
		return ""
	}
	var timestamps []float64
	for _, m := range measurements {
		timestamps = append(timestamps, float64(m.Timestamp.Unix()))
	}
	sort.Float64s(timestamps)
	newest := int64(timestamps[len(timestamps)-1])
	return strconv.FormatInt(newest, 16)
}

func toJSON(measurements map[string]measurement.Measurement, loc *time.Location) map[string]jsonMeasurement {
	mm := make(map[string]jsonMeasurement)
	for name, m := range measurements {
		mm[name] = toJSONMeasurement(m, loc)
	}
	return mm
}

func toJSONMeasurement(m measurement.Measurement, loc *time.Location) jsonMeasurement {
	ts := m.Timestamp.In(loc)
	return jsonMeasurement{
		Timestamp:   ts.Format(time.RFC3339),
		Temperature: m.Temperature,
		Humidity:    m.Humidity,
		Pressure:    m.Pressure,
	}
}
