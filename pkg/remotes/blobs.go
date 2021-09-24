package remotes

import (
	"context"
	"io"
	"net/http"
)

type blob struct {
	Service
}

var _ Object = (*blob)(nil)

func (b blob) Push(ctx context.Context, client Client, content io.ReadCloser) error {
	api, err := b.Blobs()
	if err != nil {
		return err
	}

	req, err := b.Request(ctx, http.MethodHead, api, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(ctx, req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	req, err = b.Request(ctx, http.MethodPut, api, content)
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

func (b blob) Fetch(ctx context.Context, client Client) (io.ReadCloser, error) {
	blobsAPI, err := b.Blobs()
	if err != nil {
		return nil, err
	}

	request, err := b.Request(ctx, http.MethodHead, blobsAPI, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	request, err = b.Request(ctx, http.MethodGet, blobsAPI, nil)
	if err != nil {
		return nil, err
	}

	content, err := client.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	return content.Body, nil
}
