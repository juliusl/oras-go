package remotes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type manifests struct {
	Service
}

func (m manifests) Fetch(ctx context.Context, client Client) (io.ReadCloser, error) {
	_, o, err := m.getManifest(ctx, client)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func(w *io.PipeWriter) {
		err := json.NewEncoder(w).Encode(o)
		if err != nil {
			w.CloseWithError(err)
		} else {
			w.Close()
		}
	}(pw)

	return pr, nil
}

func (m manifests) Push(ctx context.Context, client Client, body io.ReadCloser) error {
	api, err := m.Manifests()
	if err != nil {
		return err
	}

	req, err := m.Request(ctx, http.MethodHead, api, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(ctx, req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	req, err = m.Request(ctx, http.MethodPut, api, body)
	if err != nil {
		return err
	}

	resp, err = client.Do(ctx, req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (m manifests) getManifest(ctx context.Context, client Client) (desc *ocispec.Descriptor, manifest *ocispec.Manifest, err error) {
	desc, body, err := m.Resolve(ctx, client)
	if err != nil {
		return nil, nil, err
	}

	manifest = &ocispec.Manifest{}
	err = json.Unmarshal(body, manifest)
	if err != nil {
		return nil, nil, err
	}

	return desc, manifest, nil
}
