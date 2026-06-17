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

	"github.com/niktheblak/ruuvitag-measurement-api/pkg/ruuvitag"
)

func latestHandler(service ruuvitag.Service, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc, err := parseLocation(r.URL.Query().Get("tz"))
		if err != nil {
			logger.LogAttrs(r.Context(), slog.LevelWarn, "Invalid timezone", slog.String("timezone", r.URL.Query().Get("tz")), slog.Any("error", err))
			http.Error(w, "Invalid timezone", http.StatusBadRequest)
			return
		}
		count := 1
		if r.URL.Query().Get("count") != "" {
			val, err := strconv.ParseInt(r.URL.Query().Get("count"), 10, 32)
			if err != nil || val < 1 {
				http.Error(w, "Invalid count", http.StatusBadRequest)
				return
			}
			count = int(val)
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
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		measurements, err := service.Latest(ctx, columns, names, count)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.LogAttrs(r.Context(), slog.LevelError, "Timeout while querying measurements", slog.Any("error", err))
			http.Error(w, "Timeout while querying measurements", http.StatusBadGateway)
			return
		case errors.Is(err, ruuvitag.ErrInvalidColumn):
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
		resp := fetchLatest(measurements, loc)
		resp.Columns = columns
		if err := json.NewEncoder(w).Encode(resp); err != nil {
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

type response struct {
	Timezone string       `json:"tz,omitempty"`
	Columns  []string     `json:"columns"`
	Series   []seriesItem `json:"series"`
}

type seriesItem struct {
	Name         string            `json:"name,omitempty"`
	MAC          string            `json:"mac"`
	Measurements []ruuvitag.Fields `json:"measurements"`
}

func fetchLatest(measurements map[string][]ruuvitag.Fields, loc *time.Location) response {
	resp := response{
		Timezone: loc.String(),
	}
	for k, ms := range measurements {
		item := seriesItem{
			Name: k,
		}
		for _, m := range ms {
			item.MAC = *m.Addr
			m.Addr = nil // suppress MAC address from the series output
			m.Name = nil // suppress name from the series output
			m.Timestamp = m.Timestamp.In(loc)
			item.Measurements = append(item.Measurements, m)
		}
		resp.Series = append(resp.Series, item)
	}
	return resp
}
