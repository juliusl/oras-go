package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/opencontainers/go-digest"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

type artifacts struct {
	artifactType string
	Service
}

func ReferrersAPI(artifactType string) ExtendedAPI {
	return func(host, namespace, locator, mediaType string, digest digest.Digest) (*url.URL, error) {
		const format = `/oras/artifacts/v1/%s/manifests/%s/referrers?artifactType=%s`

		if digest != "" {
			locator = digest.String()
		}

		api := fmt.Sprint(format, namespace, locator, artifactType)

		return url.Parse(api)
	}
}

func (a artifacts) Resolve(ctx context.Context, client Client) (*artifactspec.Descriptor, *artifactspec.Manifest, error) {
	desc, mbytes, err := a.Service.Resolve(ctx, client)
	if err != nil {
		return nil, nil, err
	}

	adesc := &artifactspec.Descriptor{
		ArtifactType: a.artifactType,
		MediaType:    desc.MediaType,
		Size:         desc.Size,
		Digest:       desc.Digest,
	}

	amanifest := &artifactspec.Manifest{}
	err = json.Unmarshal(mbytes, amanifest)
	if err != nil {
		return nil, nil, err
	}

	return adesc, amanifest, nil
}

func (a artifacts) discover(ctx context.Context, client Client) (*Artifacts, error) {
	request, err := a.Extension(ctx, http.MethodGet, nil, ReferrersAPI(a.artifactType))
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	art := &Artifacts{}
	err = json.NewDecoder(resp.Body).Decode(art)
	if err != nil {
		return nil, err
	}

	return art, nil
}
