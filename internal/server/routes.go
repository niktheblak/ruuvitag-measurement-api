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
		if n == 1 {
			response := make(map[string]map[string]any)
			for k, m := range measurements {
				if len(m) == 0 {
					continue
				}
				response[k] = createResponse(m[0], columnMap, loc)
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.LogAttrs(r.Context(), slog.LevelError, "Error while writing output", slog.Any("error", err))
				return
			}
		} else {
			response := make(map[string][]map[string]any)
			for k, ms := range measurements {
				for _, m := range ms {
					response[k] = append(response[k], createResponse(m, columnMap, loc))
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

func createResponse(d sensor.Fields, columns map[string]string, loc *time.Location) map[string]any {
	m := make(map[string]any)
	m[columns["time"]] = d.Timestamp.In(loc)
	if c, ok := columns["mac"]; ok && d.Addr != nil {
		m[c] = *d.Addr
	}
	if c, ok := columns["name"]; ok && d.Name != nil {
		m[c] = *d.Name
	}
	if c, ok := columns["temperature"]; ok && d.Temperature != nil {
		m[c] = *d.Temperature
	}
	if c, ok := columns["humidity"]; ok && d.Humidity != nil {
		m[c] = *d.Humidity
	}
	if c, ok := columns["pressure"]; ok && d.Pressure != nil {
		m[c] = *d.Pressure
	}
	if c, ok := columns["battery_voltage"]; ok && d.BatteryVoltage != nil {
		m[c] = *d.BatteryVoltage
	}
	if c, ok := columns["tx_power"]; ok && d.TxPower != nil {
		m[c] = *d.TxPower
	}
	if c, ok := columns["acceleration_x"]; ok && d.AccelerationX != nil {
		m[c] = *d.AccelerationX
	}
	if c, ok := columns["acceleration_y"]; ok && d.AccelerationY != nil {
		m[c] = *d.AccelerationY
	}
	if c, ok := columns["acceleration_z"]; ok && d.AccelerationZ != nil {
		m[c] = *d.AccelerationZ
	}
	if c, ok := columns["movement_counter"]; ok && d.MovementCounter != nil {
		m[c] = *d.MovementCounter
	}
	if c, ok := columns["measurement_number"]; ok && d.MeasurementNumber != nil {
		m[c] = *d.MeasurementNumber
	}
	return m
}
