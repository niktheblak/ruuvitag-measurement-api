package server

import (
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
	}
	measurements, err := s.service.Current(r.Context())
	if err != nil {
		log.Printf("Error while reading response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	loc, err := location(r.URL.Query().Get("tz"))
	if err != nil {
		log.Printf("Invalid location %s: %v", r.URL.Query().Get("tz"), err)
		w.WriteHeader(http.StatusBadRequest)
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
