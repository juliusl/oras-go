package shell

import (
	"oras.land/oras-go/pkg/auth"
	"oras.land/oras-go/pkg/remotes/shell"
)

func LoginWithOpts(options ...auth.LoginOption) error {
	settings := &auth.LoginSettings{}
	for _, option := range options {
		option(settings)
	}

	ap, err := shell.ConfigureAccessProvider(".orasrc")
	if err != nil {
		return err
	}

	ctx := settings.Context
	// If wanted to try and keep backwards compat, should have some sort of implementation here
	// if settings.Hostname != "" && settings.Username != "" && settings.Secret != "" {
	//
	// }

	s, err := ap.CheckAccess(ctx, settings.Hostname, settings.Username)
	if err != nil {
		return err
	}

	return s.Error
}
