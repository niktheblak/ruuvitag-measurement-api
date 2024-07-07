package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/niktheblak/temperature-api/pkg/measurement"
)

func addRoutes(mux *http.ServeMux, service measurement.Service, logger *slog.Logger) {
	mux.Handle("/", currentHandler(service, logger))
}

func currentHandler(service measurement.Service, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc, err := location(r.URL.Query().Get("tz"))
		if err != nil {
			logger.LogAttrs(r.Context(), slog.LevelWarn, "Invalid timezone", slog.String("timezone", r.URL.Query().Get("tz")), slog.Any("error", err))
			http.Error(w, "Invalid timezone", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		measurements, err := service.Current(ctx, loc)
		if err != nil {
			logger.LogAttrs(r.Context(), slog.LevelError, "Error while getting measurements", slog.Any("error", err))
			http.Error(w, "Error while getting measurements", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		if err := json.NewEncoder(w).Encode(measurements); err != nil {
			logger.LogAttrs(r.Context(), slog.LevelError, "Error while writing output", slog.Any("error", err))
			return
		}
	})
}

func location(tz string) (loc *time.Location, err error) {
	if tz != "" {
		loc, err = time.LoadLocation(tz)
		return
	}
	loc = time.UTC
	return
}
