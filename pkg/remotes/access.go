package remotes

import (
	"context"
	"net/http"
)

type (
	AccessProvider interface {
		CheckAccess(ctx context.Context, host, username string) (*AccessStatus, error)

		RevokeAccess(ctx context.Context, host, username string) (*AccessStatus, error)

		GetAccess(ctx context.Context, challenge *AuthChallengeError) (Access, error)
	}

	Access interface {
		GetClient(ctx context.Context) (*http.Client, error)
	}

	AccessStatus struct {
		AccessProviderDir string
		UserKey           string
		TokenKey          string
	}
)
