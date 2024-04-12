package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/niktheblak/ruuvitag-common/pkg/sensor"

	"github.com/niktheblak/temperature-api/pkg/auth"
	"github.com/niktheblak/temperature-api/pkg/measurement"
	"github.com/niktheblak/temperature-api/pkg/middleware"
)

type Server struct {
	service measurement.Service
	mux     *http.ServeMux
	logger  *slog.Logger
}

func New(service measurement.Service, authenticator auth.Authenticator, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	srv := &Server{
		service: service,
		mux:     http.NewServeMux(),
		logger:  logger,
	}
	srv.routes(authenticator)
	return srv
}

func (s *Server) routes(authenticator auth.Authenticator) {
	s.mux.Handle("/", middleware.Authenticator(http.HandlerFunc(s.Current), authenticator))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Current(w http.ResponseWriter, r *http.Request) {
	loc, err := location(r.URL.Query().Get("tz"))
	if err != nil {
		s.logger.LogAttrs(r.Context(), slog.LevelWarn, "Invalid timezone", slog.String("timezone", r.URL.Query().Get("tz")), slog.Any("error", err))
		http.Error(w, "Invalid timezone", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	measurements, err := s.service.Current(ctx)
	if err != nil {
		s.logger.LogAttrs(r.Context(), slog.LevelError, "Error while getting measurements", slog.Any("error", err))
		http.Error(w, "Error while getting measurements", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	js := make(map[string]interface{})
	for name, m := range measurements {
		ts := m.Timestamp.In(loc)
		js[name] = sensor.Data{
			Timestamp:         ts,
			Addr:              m.Addr,
			Name:              m.Name,
			Temperature:       m.Temperature,
			Humidity:          m.Humidity,
			Pressure:          m.Pressure,
			DewPoint:          m.DewPoint,
			BatteryVoltage:    m.BatteryVoltage,
			TxPower:           m.TxPower,
			AccelerationX:     m.AccelerationX,
			AccelerationY:     m.AccelerationY,
			AccelerationZ:     m.AccelerationZ,
			MovementCounter:   m.MovementCounter,
			MeasurementNumber: m.MeasurementNumber,
		}
	}
	if err := json.NewEncoder(w).Encode(js); err != nil {
		s.logger.LogAttrs(r.Context(), slog.LevelError, "Error while writing output", slog.Any("error", err))
		return
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
