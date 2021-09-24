package remotes

import (
	"context"

	"github.com/containerd/containerd/content"
	ctrdremotes "github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	orasRemotes "oras.land/oras-go/pkg/remotes"
)

func NewDiscoverFetchResolver(ctx context.Context, registry *orasRemotes.Registry, reference string) (ctrdremotes.Resolver, error) {
	funcs := registry.Adapter()

	return DiscoverFetch(ctx, funcs.Fetcher(), funcs.Resolver(), funcs.Discoverer(), reference)
}

func NewPushPullResolver(ctx context.Context, registry *orasRemotes.Registry, writer content.Writer, desc ocispec.Descriptor) (ctrdremotes.Resolver, error) {
	funcs := registry.Adapter()

	return PushPull(ctx, funcs.Fetcher(), containerdPusher(nil), funcs.Resolver(), funcs.Discoverer(), desc)
}
