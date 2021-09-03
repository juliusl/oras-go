package remotes

import (
	"context"
	"net/http"
)

type (
	AccessProvider interface {
		GetAccess(ctx context.Context, challenge *AuthChallengeError) (Access, error)
	}

	Access interface {
		GetClient(ctx context.Context) (*http.Client, error)
	}
)
