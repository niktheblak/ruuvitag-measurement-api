package server

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/niktheblak/temperature-api/pkg/auth"
	"github.com/niktheblak/temperature-api/pkg/measurement"
	"github.com/niktheblak/temperature-api/pkg/middleware"
)

func New(service measurement.Service, authenticator auth.Authenticator, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	mux := http.NewServeMux()
	addRoutes(mux, service, logger)
	var handler http.Handler = mux
	handler = middleware.Authenticator(handler, authenticator)
	return handler
}
