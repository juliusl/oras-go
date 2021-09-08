package shell

import (
	"context"
	"encoding/json"
	"os/exec"

	"oras.land/oras-go/pkg/remotes"
)

func ConfigureAccessProvider(path string) (remotes.AccessProvider, error) {
	return &accessProvider{
		path:         path,
		accessStatus: nil,
	}, nil
}

type accessProvider struct {
	path         string
	accessStatus *remotes.AccessStatus
}

func (s *accessProvider) CheckAccess(ctx context.Context, host, username string) (*remotes.AccessStatus, error) {
	status := exec.Command(s.path, "status", host, username)

	out, err := status.Output()
	if err != nil {
		return nil, err
	}

	st := &remotes.AccessStatus{} // TODO: Could cache this
	err = json.Unmarshal(out, st)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (s *accessProvider) RevokeAccess(ctx context.Context, host, username string) (*remotes.AccessStatus, error) {
	status := exec.Command(s.path, "revoke", host, username)

	out, err := status.Output()
	if err != nil {
		return nil, err
	}

	st := &remotes.AccessStatus{}
	err = json.Unmarshal(out, st)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (s *accessProvider) GetAccess(ctx context.Context, challenge *remotes.AuthChallengeError) (remotes.Access, error) {
	realm, service, scope, ns, err := challenge.ParseChallenge()
	if err != nil {
		return nil, err
	}

	status := exec.Command(s.path, "challenge", realm, service, scope)

	out, err := status.Output()
	if err != nil {
		return nil, err
	}

	st := &remotes.AccessStatus{}
	err = json.Unmarshal(out, st)
	if err != nil {
		return nil, err
	}

	a, err := remotes.FromDirectory(ctx, st.AccessProviderDir, ns, scope, st.UserKey, st.TokenKey)
	if err != nil {
		return nil, err
	}

	return a, nil
}
