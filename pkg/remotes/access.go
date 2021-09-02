package remotes

import (
	"context"
	"net/http"
)

type (
	AccessProvider interface {
		GetClient(ctx context.Context) (*http.Client, error)
	}
)
