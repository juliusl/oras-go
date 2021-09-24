package remotes

import (
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

//
func (r *Registry) Locate(ref, mediatype string, digest digest.Digest) (Service, error) {
	_, host, ns, loc, err := Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("reference is is not valid")
	}

	if ns != r.namespace {
		return nil, fmt.Errorf("namespace does not match current registry context")
	}

	if host != r.host {
		return nil, fmt.Errorf("host does not match current registry context")
	}

	// format the reference
	return &reference{
		add: address{
			host: r.host,
			ns:   r.namespace,
			loc:  loc,
		},
		media:  mediatype,
		digest: digest,
	}, nil
}

// Query is a function that queries the registry for a reference using the provided parameters
// before the query is made, we validate that the reference can be queried from this registry
func (r *Registry) Query(ctx context.Context, ref, accept string, digest digest.Digest) (*ocispec.Descriptor, *ocispec.Manifest, error) {
	srv, err := r.Locate(ref, accept, digest)
	if err != nil {
		return nil, nil, err
	}

	m := manifests{Service: srv}

	desc, manifest, err := m.getManifest(ctx, r)
	if err != nil {
		return nil, nil, err
	}

	return desc, manifest, nil
}

func (r *Registry) FindManifest(ctx context.Context, ref, accept string) (*ocispec.Descriptor, *ocispec.Manifest, error) {
	return r.Query(ctx, ref, accept, "")
}

func (r *Registry) GetManifest(ctx context.Context, ref, accept string, digest digest.Digest) (*ocispec.Descriptor, *ocispec.Manifest, error) {
	return r.Query(ctx, ref, accept, digest)
}
