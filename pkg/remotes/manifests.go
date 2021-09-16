package remotes

import (
	"context"
	"encoding/json"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type manifests struct {
	ref reference
}

// getDescriptor tries to resolve the reference to a descriptor using the headers
func (m manifests) getDescriptor(ctx context.Context, doer Doer) (ocispec.Descriptor, error) {
	request, err := endpoints.e3HEAD.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	resp, err := doer.Do(ctx, request)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	defer resp.Body.Close()

	d := resp.Header.Get("Docker-Content-Digest")
	c := resp.Header.Get("Content-Type")
	s := resp.ContentLength

	err = digest.Digest(d).Validate()
	if err == nil && c != "" && s > 0 {
		// TODO: Write annotations
		return ocispec.Descriptor{
			Digest:    digest.Digest(d),
			MediaType: c,
			Size:      s,
		}, nil
	}

	return ocispec.Descriptor{}, err
}

// getDescriptorWithManifest tries to resolve the reference by fetching the manifest
func (m manifests) getDescriptorWithManifest(ctx context.Context, doer Doer) (*ocispec.Manifest, digest.Digest, error) {
	// If we didn't get a digest by this point, we need to pull the manifest
	request, err := endpoints.e3GET.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return nil, "", err
	}

	resp, err := doer.Do(ctx, request)
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()

	manifest := &ocispec.Manifest{}
	err = json.NewDecoder(resp.Body).Decode(manifest)
	if err != nil {
		return nil, "", err
	}

	d := resp.Header.Get("Docker-Content-Digest")

	return manifest, digest.Digest(d), nil
}

func (m manifests) getManifest(ctx context.Context, doer Doer) (*ocispec.Manifest, error) {
	// If we didn't get a digest by this point, we need to pull the manifest
	request, err := endpoints.e3GET.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return nil, err
	}

	resp, err := doer.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	manifest := &ocispec.Manifest{}
	err = json.NewDecoder(resp.Body).Decode(manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}
