package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/niktheblak/temperature-api/pkg/measurement"
)

type Server struct {
	service measurement.Service
	mux     *http.ServeMux
}

func New(service measurement.Service) *Server {
	srv := &Server{
		service: service,
		mux:     http.NewServeMux(),
	}
	srv.routes()
	return srv
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.Current)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Current(w http.ResponseWriter, r *http.Request) {
	type meas struct {
		Timestamp   string  `json:"ts"`
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
		Pressure    float64 `json:"pressure"`
		DewPoint    float64 `json:"dew_point"`
	}
	loc, err := location(r.URL.Query().Get("tz"))
	if err != nil {
		log.Printf("Invalid location %s: %v", r.URL.Query().Get("tz"), err)
		http.Error(w, "Invalid location", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	measurements, err := s.service.Current(ctx)
	if err != nil {
		log.Printf("Error while getting measurements: %v", err)
		http.Error(w, "Error while getting measurements", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	js := make(map[string]interface{})
	for name, m := range measurements {
		ts := m.Timestamp.In(loc)
		js[name] = meas{
			Timestamp:   ts.Format(time.RFC3339),
			Temperature: m.Temperature,
			Humidity:    m.Humidity,
			Pressure:    m.Pressure,
			DewPoint:    m.DewPoint,
		}
	}
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
