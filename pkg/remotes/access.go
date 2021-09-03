package remotes

import (
	"context"
	"net/http"
)

type (
	AccessProvider interface {
		GetAccess(challenge *AuthChallengeError) (Access, error)
	}

	Access interface {
		GetClient(ctx context.Context) (*http.Client, error)
	}
)
