package middleware

import (
	"net/http"
	"strings"

	"github.com/niktheblak/temperature-api/pkg/auth"
)

func Authenticator(handler http.Handler, authenticator auth.Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		err := authenticator.Authenticate(r.Context(), token)
		if err != nil {
			forbidden(w)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func forbidden(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}
