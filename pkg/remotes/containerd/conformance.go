package remotes

import (
	ctrRemotes "github.com/containerd/containerd/remotes"
)

// Ensure the interfaces still match
var (
	_ ctrRemotes.Resolver = (*resolver)(nil)
	_ ctrRemotes.Fetcher  = (*resolver)(nil)
	_ ctrRemotes.Pusher   = (*resolver)(nil)
)
