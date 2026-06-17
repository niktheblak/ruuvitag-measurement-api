package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/niktheblak/ruuvitag-common/pkg/columnmap"
	"github.com/niktheblak/ruuvitag-common/pkg/sensor"

	"github.com/niktheblak/ruuvitag-measurement-api/pkg/ruuvitag"
)

func latestHandler(service ruuvitag.Service, columnMap map[string]string, logger *slog.Logger) http.Handler {
	defaultColumns := []string{columnMap["time"], columnMap["mac"]}
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
		if slices.Contains(columns, columnMap["name"]) {
			http.Error(w, "Query must not contain colum \"name\"", http.StatusBadRequest)
			return
		}
		names, err := parseCSV(r.URL.Query().Get("names"))
		if err != nil {
			http.Error(w, "Invalid names", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		queryColumns := mergeColumns(columns, defaultColumns)
		measurements, err := service.Latest(ctx, count, queryColumns, names)
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
		resp := getLatest(measurements, count, columnMap, loc)
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
	Name         string           `json:"name,omitempty"`
	MAC          string           `json:"mac"`
	Measurements []map[string]any `json:"measurements"`
}

func getLatest(measurements map[string][]sensor.Fields, count int, columnMap map[string]string, loc *time.Location) response {
	resp := response{
		Timezone: loc.String(),
	}
	for k, ms := range measurements {
		item := seriesItem{
			Name: k,
		}
		for i, m := range ms {
			if i == count {
				break
			}
			if m.Addr == nil {
				panic("Mandatory field m.Addr is nil")
			}
			item.MAC = *m.Addr
			m.Timestamp = m.Timestamp.In(loc)
			fields := columnmap.TransformFields(columnMap, m)
			delete(fields, columnMap["mac"])
			item.Measurements = append(item.Measurements, fields)
		}
		resp.Series = append(resp.Series, item)
	}
	return resp
}

func mergeColumns(a, b []string) []string {
	m := make(map[string]interface{})
	for _, c := range a {
		m[c] = struct{}{}
	}
	for _, c := range b {
		m[c] = struct{}{}
	}
	var r []string
	for k := range m {
		r = append(r, k)
	}
	return r
}
