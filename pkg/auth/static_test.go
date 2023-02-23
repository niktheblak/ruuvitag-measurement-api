package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticAuthenticator_Authenticate(t *testing.T) {
	t.Parallel()

	a := Static("test_tkn_f12321")
	t.Run("Authenticated", func(t *testing.T) {
		t.Parallel()

		err := a.Authenticate(context.Background(), "test_tkn_f12321")
		assert.NoError(t, err)
	})
	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()

		err := a.Authenticate(context.Background(), "another_tkn_cf7a6")
		assert.ErrorIs(t, err, ErrNotAuthorized)
	})
}
