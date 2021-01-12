package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/niktheblak/temperature-api/pkg/measurement"
)

type Server struct {
	Service measurement.Service
	Router  *httprouter.Router
}

func (s *Server) Routes() {
	s.Router.HandlerFunc("GET", "/ready", s.Ready())
	s.Router.HandlerFunc("GET", "/health", s.Health())
	s.Router.HandlerFunc("GET", "/", s.Current())
}

func (s *Server) Current() http.HandlerFunc {
	type meas struct {
		Timestamp   string  `json:"ts"`
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
		Pressure    float64 `json:"pressure"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
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
}

func (s *Server) Ready() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := map[string]interface{}{
			"status": "ok",
		}
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.Fatal(err)
		}
	}
}

func (s *Server) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		err := s.Service.Ping(ctx)
		cancel()
		if err == nil {
			w.WriteHeader(http.StatusOK)
			status := map[string]interface{}{
				"status": "ok",
			}
			if err := json.NewEncoder(w).Encode(status); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("Error during status check: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			status := map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
			if err := json.NewEncoder(w).Encode(status); err != nil {
				log.Fatal(err)
			}
		}
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
