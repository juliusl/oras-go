package shell

import (
	"oras.land/oras-go/pkg/auth"
	"oras.land/oras-go/pkg/remotes/shell"
)

func NewLogin(rcPath string) *ShellLogin {
	return &ShellLogin{
		rcPath: rcPath,
	}
}

type ShellLogin struct {
	rcPath string
}

func (s *ShellLogin) LoginWithOpts(options ...auth.LoginOption) error {
	settings := &auth.LoginSettings{}
	for _, option := range options {
		option(settings)
	}

	ap, err := shell.ConfigureAccessProvider(s.rcPath)
	if err != nil {
		return err
	}

	ctx := settings.Context
	status, err := ap.CheckAccess(ctx, settings.Hostname, settings.Username)
	if err != nil {
		return err
	}

	return status.Error
}
