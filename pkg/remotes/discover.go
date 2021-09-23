package remotes

import (
	"context"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

type Artifacts struct {
	References []artifactspec.Descriptor `json:"references"` // References is an array of descriptors that point to a manifest
}

type DiscoverFunc func(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error)

type Discoverer interface {
	Discover(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error)
}
