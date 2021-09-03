package remotes

import (
	"context"
	"io"
)

type blob struct {
	ref reference
}

// push
func (b blob) push(ctx context.Context) error {
	// First write the manifest

	// Return a writer to push the content
	return nil
}

func (b blob) fetch(ctx context.Context, doer Doer) (io.ReadCloser, error) {
	request, err := endpoints.e2HEAD.prepareWithDescriptor()(ctx,
		b.ref.add.host,
		b.ref.add.ns,
		b.ref.digst.String(),
		b.ref.media)

	if err != nil {
		return nil, err
	}

	resp, err := doer.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	request, err = endpoints.e2GET.prepareWithDescriptor()(ctx,
		b.ref.add.host,
		b.ref.add.ns,
		b.ref.digst.String(),
		b.ref.media)
	if err != nil {
		return nil, err
	}

	content, err := doer.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	return content.Body, nil
}
