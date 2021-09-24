package remotes

import (
	"context"
	"fmt"
	"io"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"oras.land/oras-go/pkg/content"
)

type CONTENT_WRITER_ALIAS = *content.OCIStore

// Adapter is an opaque type that provides an adapter layer to be backwards compatible with
// containerd's remote.Resolver
type Adapter struct {
	fetcher    func(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error)
	resolver   func(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error)
	discoverer func(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error)
}

func (r *Registry) Adapter() *Adapter {
	return &Adapter{
		fetcher:    r.fetch,
		resolver:   r.resolve,
		discoverer: r.discover,
	}
}

func (r *Adapter) Fetcher() func(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	return r.fetcher
}

func (r *Adapter) Resolver() func(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	return r.resolver
}

func (r *Adapter) Discoverer() func(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error) {
	return r.discoverer
}

const (
	manifestV2json     string = "application/vnd.docker.distribution.manifest.v2+json"
	manifestlistV2json string = "application/vnd.docker.distribution.manifest.list.v2+json"
)

// resolve resolves a reference to a descriptor
func (r *Registry) resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	if r == nil {
		return "", ocispec.Descriptor{}, fmt.Errorf("reference to registry is nil")
	}

	accept := strings.Join([]string{
		manifestV2json,
		manifestlistV2json,
		ocispec.MediaTypeImageManifest,
		artifactspec.MediaTypeArtifactManifest,
		"*/*",
	}, ",")

	d, _, err := r.FindManifest(ctx, ref, accept)
	if err != nil {
		return "", ocispec.Descriptor{}, err
	}

	return ref, *d, nil
}

func (r *Registry) fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	if r == nil {
		return nil, fmt.Errorf("reference to registry is nil")
	}

	return blob{
		Service: reference{
			add: address{
				host: r.host,
				ns:   r.namespace,
				loc:  "",
			},
			digest: desc.Digest,
			media:  desc.MediaType,
		},
	}.Fetch(ctx, r)
}

func (r *Registry) discover(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error) {
	if r == nil {
		return nil, fmt.Errorf("reference is nil")
	}

	return artifacts{
		artifactType: artifactType,
		Service: reference{
			add: address{
				host: r.host,
				ns:   r.namespace,
				loc:  "",
			},
			digest: desc.Digest,
			media:  desc.MediaType,
		},
	}.discover(ctx, r)
}
