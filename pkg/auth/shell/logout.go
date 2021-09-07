package shell

import (
	"context"

	"oras.land/oras-go/pkg/remotes/shell"
)

func (s *ShellLogin) Logout(ctx context.Context, hostname string) error {
	ap, err := shell.ConfigureAccessProvider(s.rcPath)
	if err != nil {
		return err
	}

	st, err := ap.RevokeAccess(ctx, hostname, "")
	if err != nil {
		return err
	}

	return st.Error
}
