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
		n, err := parseN(r.URL.Query().Get("n"), 1)
		if err != nil {
			http.Error(w, "Invalid n", http.StatusBadRequest)
			return
		}
		columns, err := parseColumns(r.URL.Query().Get("columns"))
		if err != nil {
			http.Error(w, "Invalid columns", http.StatusBadRequest)
			return
		}
		logger.LogAttrs(r.Context(), slog.LevelDebug, "Columns from query", slog.Any("columns", columns))
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		measurements, err := service.Latest(ctx, n, columns)
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
		if n == 1 {
			response := make(map[string]map[string]any)
			for k, m := range measurements {
				if len(m) == 0 {
					continue
				}
				fields := m[0]
				fields.Timestamp = fields.Timestamp.In(loc)
				response[k] = columnmap.TransformFields(columnMap, fields)
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.LogAttrs(r.Context(), slog.LevelError, "Error while writing output", slog.Any("error", err))
				return
			}
		} else {
			response := make(map[string][]map[string]any)
			for k, ms := range measurements {
				for _, m := range ms {
					m.Timestamp = m.Timestamp.In(loc)
					response[k] = append(response[k], columnmap.TransformFields(columnMap, m))
				}
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.LogAttrs(r.Context(), slog.LevelError, "Error while writing output", slog.Any("error", err))
				return
			}
		}
	})
}

func parseN(n string, defaultValue int) (int, error) {
	if n == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(n)
}

func parseLocation(tz string) (loc *time.Location, err error) {
	if tz != "" {
		loc, err = time.LoadLocation(tz)
		return
	}
	loc = time.UTC
	return
}

func parseColumns(columns string) ([]string, error) {
	if columns == "" {
		return nil, nil
	}
	r := regexp.MustCompile(`^[\w,]*\w$`)
	if !r.MatchString(columns) {
		return nil, fmt.Errorf("invalid columns: %s", columns)
	}
	return strings.Split(columns, ","), nil
}
