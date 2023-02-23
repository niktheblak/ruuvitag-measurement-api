package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlwaysAllowAuthenticator_Authenticate(t *testing.T) {
	t.Parallel()

	a := AlwaysAllow()
	err := a.Authenticate(context.Background(), "any_token")
	assert.NoError(t, err)
	err = a.Authenticate(context.Background(), "")
	assert.NoError(t, err)
}
