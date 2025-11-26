package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/niktheblak/ruuvitag-measurement-api/pkg/ruuvitag"
	"github.com/niktheblak/web-common/pkg/auth"
	"github.com/niktheblak/web-common/pkg/healthcheck"
	"github.com/niktheblak/web-common/pkg/middleware"
)

func New(service ruuvitag.Service, columns map[string]string, authenticator auth.Authenticator, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	mux := http.NewServeMux()
	mux.Handle("/health", healthcheck.HealthCheck(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return service.Ping(ctx)
	}, logger))
	mux.Handle("/", middleware.Authenticator(latestHandler(service, columns, logger), authenticator))
	return mux
}
