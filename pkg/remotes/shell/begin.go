package shell

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"

	"oras.land/oras-go/pkg/remotes"
	"oras.land/oras-go/pkg/remotes/oauth"
)

func FromDirectory(ctx context.Context, accessDirectory, namespace, scopes, userkey, tokenkey string) (remotes.Access, error) {
	// TODO document tokens interface
	tokenssh := path.Join(accessDirectory, namespace, "tokens")
	if tokenssh == "" {
		return nil, errors.New("remotes: could not create path for tokens interface")
	}

	// tokenssh is an interface for getting token data
	_, err := os.Stat(tokenssh)
	if err != nil {
		return nil, err
	}

	// First reverse lookup the userkey for the actual username
	c := exec.Command(tokenssh, "get-user", userkey)
	if c == nil {
		return nil, errors.New("could not create command")
	}

	out, err := c.Output()
	if err != nil {
		return nil, err
	}

	user := string(out)

	// Once the real username is resolved, lookup the resolved token
	c = exec.Command(tokenssh, "get-access-token", user, tokenkey)
	if c == nil {
		return nil, errors.New("could not create command")
	}

	out, err = c.Output()
	if err != nil {
		return nil, err
	}

	token := string(out)
	ts := oauth.NewBasicAuthTokenSource(ctx, namespace, user, token, scopes)
	return oauth.NewTokenSourceAccess(ts), nil
}
