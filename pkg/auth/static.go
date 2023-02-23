package auth

import (
	"context"
	"crypto/subtle"
)

type StaticAuthenticator struct {
	AllowedTokens []string
}

func (a *StaticAuthenticator) Authenticate(ctx context.Context, token string) error {
	for _, t := range a.AllowedTokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(t)) == 1 {
			return nil
		}
	}
	return ErrNotAuthorized
}

func Static(tokens ...string) *StaticAuthenticator {
	return &StaticAuthenticator{
		AllowedTokens: tokens,
	}
}
