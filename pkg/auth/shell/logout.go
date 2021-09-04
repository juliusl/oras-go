package shell

import (
	"context"

	"oras.land/oras-go/pkg/remotes/shell"
)

func Logout(ctx context.Context, hostname string) error {
	ap, err := shell.ConfigureAccessProvider(".orasrc")
	if err != nil {
		return err
	}

	st, err := ap.RevokeAccess(ctx, hostname, "")
	if err != nil {
		return err
	}

	return st.Error
}
