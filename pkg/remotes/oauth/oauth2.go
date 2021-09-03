package oauth

import (
	"context"
	"fmt"
	"regexp"

	"oras.land/oras-go/pkg/remotes"
)

// OAuth2Provider is the concrete implementation of receiving a challenge and returning an access provider
type OAuth2Provider = func(ctx context.Context, realm, service, scope string) (remotes.AccessProvider, error)

// Challenge header examples...
// Www-Authenticate: Bearer realm="https://example.azurecr.io/oauth2/token",service="example.azurecr.io"
// Www-Authenticate: Bearer realm="https://example.azurecr.io/oauth2/token",service="example.azurecr.io",scope="repository:ubuntu:pull"
// Www-Authenticate: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:samalba/my-app:pull,push"

// Parsing headers to format requests to OAuth2Providers
var parseBearerChallengeHeader = regexp.MustCompile(`Www-Authenticate:.Bearer.realm="(.*)",service="(.*)"`)
var parseBearerChallengeHeaderWithScope = regexp.MustCompile(`Www-Authenticate:.Bearer.realm="(.*)",service="(.*)",scope="(.*)`)
var parseNamespaceFromScope = regexp.MustCompile(`repository:(.*):`)

// NewRegistryWithOAuth2
func NewRegistryWithOAuth2(ctx context.Context, challenge string, providers []OAuth2Provider) (*remotes.Registry, error) {
	var (
		realm, service, scope string
		namespace             string
	)

	results := parseBearerChallengeHeaderWithScope.FindAllStringSubmatch(challenge, -1)
	if len(results) <= 0 {
		results = parseBearerChallengeHeader.FindAllStringSubmatch(challenge, -1)
		if len(results) <= 0 {
			return nil, fmt.Errorf("invalid challenge")
		}
	}

	realm = results[0][1]
	service = results[0][2]
	if len(results[0]) > 3 {
		scope = results[0][3]
	}

	for _, p := range providers {
		access, err := p(ctx, realm, service, scope)
		if err != nil || access == nil {
			continue
		}

		if scope != "" {
			results = parseNamespaceFromScope.FindAllStringSubmatch(scope, -1)
			if len(results) > 0 && len(results[0]) > 1 {
				namespace = results[0][1]
			}
		}

		return remotes.NewRegistry(service, namespace, access), nil
	}

	return nil, fmt.Errorf("could not find an access provider for registry with challenge %s", challenge)
}
