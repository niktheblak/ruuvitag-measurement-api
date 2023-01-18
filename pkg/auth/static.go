package auth

import (
	"context"
)

type StaticAuthenticator struct {
	AllowedTokens []string
}

func (a *StaticAuthenticator) Authenticate(ctx context.Context, token string) error {
	for _, t := range a.AllowedTokens {
		if token == t {
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
