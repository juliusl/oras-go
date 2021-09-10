package shell

import (
	"oras.land/oras-go/pkg/auth"
	remotessh "oras.land/oras-go/pkg/remotes/shell"
)

func NewLogin(loginDir string) *ShellLogin {
	return &ShellLogin{
		LoginDir: loginDir,
	}
}

type ShellLogin struct {
	LoginDir          string
	AccessProviderDir string
}

func (s *ShellLogin) LoginWithOpts(options ...auth.LoginOption) error {
	settings := &auth.LoginSettings{}
	for _, option := range options {
		option(settings)
	}

	ap, err := remotessh.ConfigureAccessProvider(s.LoginDir, s.AccessProviderDir)
	if err != nil {
		return err
	}

	ctx := settings.Context
	status, err := ap.CheckAccess(ctx, settings.Hostname, settings.Username)
	if err != nil {
		return err
	}

	s.AccessProviderDir = status.AccessRoot

	return nil
}
