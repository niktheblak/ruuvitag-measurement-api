package auth

import (
	"context"
	"errors"
)

var ErrNotAuthorized = errors.New("not authorized")

type Authenticator interface {
	Authenticate(ctx context.Context, token string) error
}

type AlwaysAllowAuthenticator struct {
}

func (a *AlwaysAllowAuthenticator) Authenticate(ctx context.Context, token string) error {
	return nil
}

func AlwaysAllow() *AlwaysAllowAuthenticator {
	return &AlwaysAllowAuthenticator{}
}
