package shell

import (
	"context"
	"os/exec"

	"oras.land/oras-go/pkg/remotes"
	"oras.land/oras-go/pkg/remotes/oauth"
)

type shellAccessProvider struct {
}

func (s *shellAccessProvider) GetAccess(ctx context.Context, challenge *remotes.AuthChallengeError) (remotes.Access, error) {
	realm, service, scope, ns, err := challenge.ParseChallenge()
	if err != nil {
		return nil, err
	}

	access := exec.Command(".orasrc", "challenge", realm, service, scope)

	out, err := access.Output()
	if err != nil {
		return nil, err
	}

	// TODO
	accessToken := string(out)

	ts := oauth.NewBasicAuthTokenSource(ctx, ns, "", accessToken, scope)

	return oauth.NewTokenSourceAccess(ts), nil
}
