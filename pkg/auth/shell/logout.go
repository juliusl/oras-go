package shell

import (
	"context"

	remotessh "oras.land/oras-go/pkg/remotes/shell"
)

func (s *ShellLogin) Logout(ctx context.Context, hostname string) error {
	ap, err := remotessh.ConfigureAccessProvider(s.LoginDir, s.AccessProviderDir)
	if err != nil {
		return err
	}

	_, err = ap.RevokeAccess(ctx, hostname, "")
	if err != nil {
		return err
	}

	return nil
}
