package shell

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"oras.land/oras-go/pkg/remotes"
)

func ConfigureAccessProvider(loginRoot, accessRoot string) (remotes.AccessProvider, error) {
	return &accessProvider{
		loginrc:      path.Join(loginRoot, "loginrc"),
		accessrc:     path.Join(accessRoot, "accessrc"),
		accessStatus: nil,
	}, nil
}

type accessProvider struct {
	loginrc      string
	accessrc     string
	accessStatus *remotes.AccessStatus
}

const anonymous = "Anonymous\n"

func (s *accessProvider) CheckAccess(ctx context.Context, host, image, username string) (*remotes.AccessStatus, error) {
	status := exec.Command(s.loginrc, "status", host, username)

	out, err := status.CombinedOutput()
	if err != nil && string(out) != anonymous {
		return nil, err
	} else if string(out) == anonymous {
		envbegin := os.Getenv("ORAS_BEGIN_ENV")
		envnamespace := os.Getenv("ORAS_NAMESPACE")
		if envbegin == "" && envnamespace == "" {
			return nil, errors.New("ORASRC environment has not been setup")
		}

		accessroot := path.Join(envbegin, envnamespace, "access")
		fi, err := os.Stat(accessroot)
		if err != nil {
			return nil, err
		}

		if !fi.IsDir() {
			return nil, errors.New("missing access root")
		}
		return &remotes.AccessStatus{
			Image:      image,
			AccessRoot: accessroot,
		}, nil
	}

	st := &remotes.AccessStatus{} // TODO: Could cache this
	err = json.Unmarshal(out, st)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (s *accessProvider) RevokeAccess(ctx context.Context, host, username string) (*remotes.AccessStatus, error) {
	status := exec.Command(s.loginrc, "revoke", host, username)

	out, err := status.CombinedOutput()
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

	status := exec.Command(s.loginrc, "challenge", realm, service, scope)
	_, err = status.CombinedOutput()
	if err != nil {
		return nil, err
	}

	statusJSON := path.Join(path.Dir(s.loginrc), service, strings.Replace(scope, ":", "/", -1), "status.json")
	f, err := ioutil.ReadFile(statusJSON)
	if err != nil {
		return nil, err
	}

	st := &remotes.AccessStatus{}
	err = json.Unmarshal(f, st)
	if err != nil {
		return nil, err
	}

	a, err := FromDirectory(ctx, st.AccessRoot, service, ns, scope, st.UserKey, st.TokenKey)
	if err != nil {
		return nil, err
	}

	return a, nil
}
