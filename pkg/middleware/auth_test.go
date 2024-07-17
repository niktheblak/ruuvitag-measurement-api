package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/niktheblak/ruuvitag-measurement-api/pkg/auth"
)

func TestAuthenticator(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})
	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()

		a := Authenticator(handler, auth.Static("test_token_2dc9a"))
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
	})
	t.Run("Authenticated", func(t *testing.T) {
		t.Parallel()

		a := Authenticator(handler, auth.Static("test_token_2dc9a"))
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer test_token_2dc9a")
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
	t.Run("Invalid token", func(t *testing.T) {
		t.Parallel()

		a := Authenticator(handler, auth.Static("test_token_2dc9a"))
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer other_token_7a3b1")
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
	})
}
