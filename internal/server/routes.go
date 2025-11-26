package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/niktheblak/ruuvitag-common/pkg/columnmap"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"
	"github.com/niktheblak/ruuvitag-measurement-api/pkg/ruuvitag"
)

func latestHandler(service ruuvitag.Service, columnMap map[string]string, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc, err := parseLocation(r.URL.Query().Get("tz"))
		if err != nil {
			logger.LogAttrs(r.Context(), slog.LevelWarn, "Invalid timezone", slog.String("timezone", r.URL.Query().Get("tz")), slog.Any("error", err))
			http.Error(w, "Invalid timezone", http.StatusBadRequest)
			return
		}
		n := 1
		if r.URL.Query().Get("n") != "" {
			val, err := strconv.ParseInt(r.URL.Query().Get("n"), 10, 32)
			if err != nil || val < 1 {
				http.Error(w, "Invalid n", http.StatusBadRequest)
				return
			}
			n = int(val)
		}
		columns, err := parseCSV(r.URL.Query().Get("columns"))
		if err != nil {
			http.Error(w, "Invalid columns", http.StatusBadRequest)
			return
		}
		names, err := parseCSV(r.URL.Query().Get("names"))
		if err != nil {
			http.Error(w, "Invalid names", http.StatusBadRequest)
			return
		}
		logger.LogAttrs(r.Context(), slog.LevelDebug, "RuuviTags and columns from query", slog.Any("columns", columns), slog.Any("names", names))
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		measurements, err := service.Latest(ctx, n, columns, names)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.LogAttrs(r.Context(), slog.LevelError, "Timeout while querying measurements", slog.Any("error", err))
			http.Error(w, "Timeout while querying measurements", http.StatusBadGateway)
			return
		case errors.Is(err, sensor.ErrInvalidColumn) || errors.Is(err, sensor.ErrMissingColumn):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case err == nil:
		default:
			logger.LogAttrs(r.Context(), slog.LevelError, "Error while getting measurements", slog.Any("error", err))
			http.Error(w, "Error while getting measurements", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		var response any
		if n == 1 {
			// flatten the map to a single measurement per column for easier JSON readability
			response = newest(measurements, columnMap, loc)
		} else {
			// return measurements as a map of lists with a maximum of n measurements per column
			response = newestN(measurements, n, columnMap, loc)
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.LogAttrs(r.Context(), slog.LevelError, "Error while writing output", slog.Any("error", err))
			return
		}
	})
}

func parseLocation(tz string) (loc *time.Location, err error) {
	if tz != "" {
		loc, err = time.LoadLocation(tz)
		return
	}
	loc = time.UTC
	return
}

func parseCSV(values string) ([]string, error) {
	if values == "" {
		return nil, nil
	}
	r := regexp.MustCompile(`^[\w\s,]*\w$`)
	if !r.MatchString(values) {
		return nil, fmt.Errorf("invalid values: %s", values)
	}
	return strings.Split(values, ","), nil
}

func newest(measurements map[string][]sensor.Fields, columnMap map[string]string, loc *time.Location) map[string]map[string]any {
	response := make(map[string]map[string]any)
	for k, m := range measurements {
		if len(m) == 0 {
			continue
		}
		fields := m[0]
		fields.Timestamp = fields.Timestamp.In(loc)
		response[k] = columnmap.TransformFields(columnMap, fields)
	}
	return response
}

func newestN(measurements map[string][]sensor.Fields, n int, columnMap map[string]string, loc *time.Location) map[string][]map[string]any {
	response := make(map[string][]map[string]any)
	for k, ms := range measurements {
		for i, m := range ms {
			if i == n {
				break
			}
			m.Timestamp = m.Timestamp.In(loc)
			response[k] = append(response[k], columnmap.TransformFields(columnMap, m))
		}
	}
	return response
}
