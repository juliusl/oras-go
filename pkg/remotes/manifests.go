package remotes

import (
	"context"
	"encoding/json"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

type manifests struct {
	ref reference
}

func (m manifests) getManifest(ctx context.Context, doer Doer) (desc *ocispec.Descriptor, manifest *ocispec.Manifest, err error) {
	request, err := endpoints.e3HEAD.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return nil, nil, err
	}

	resp, err := doer.Do(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	d := resp.Header.Get("Docker-Content-Digest")
	c := resp.Header.Get("Content-Type")
	s := resp.ContentLength

	err = digest.Digest(d).Validate()
	if err != nil {
		return nil, nil, err
	}

	desc = &ocispec.Descriptor{
		Digest:    digest.Digest(d),
		MediaType: c,
		Size:      s,
	}

	// If we didn't get a digest by this point, we need to pull the manifest
	request, err = endpoints.e3GET.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return nil, nil, err
	}

	resp, err = doer.Do(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	manifest = &ocispec.Manifest{}
	err = json.NewDecoder(resp.Body).Decode(manifest)
	if err != nil {
		return nil, nil, err
	}

	return desc, manifest, nil
}

func (m manifests) getArtifactManifest(ctx context.Context, doer Doer) (desc *artifactspec.Descriptor, manifest *artifactspec.Manifest, err error) {
	request, err := endpoints.e3HEAD.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return nil, nil, err
	}

	resp, err := doer.Do(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	d := resp.Header.Get("Docker-Content-Digest")
	c := resp.Header.Get("Content-Type")
	s := resp.ContentLength

	err = digest.Digest(d).Validate()
	if err != nil {
		return nil, nil, err
	}

	desc = &artifactspec.Descriptor{
		Digest:    digest.Digest(d),
		MediaType: c,
		Size:      s,
	}

	// If we didn't get a digest by this point, we need to pull the manifest
	request, err = endpoints.e3GET.prepare()(ctx,
		m.ref.add.host,
		m.ref.add.ns,
		m.ref.add.loc)
	if err != nil {
		return nil, nil, err
	}

	resp, err = doer.Do(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	manifest = &artifactspec.Manifest{}
	err = json.NewDecoder(resp.Body).Decode(manifest)
	if err != nil {
		return nil, nil, err
	}

	return desc, manifest, nil
}
