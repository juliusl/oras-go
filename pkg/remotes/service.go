package remotes

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type (
	// Service is an interface fixed to a single reference
	Service interface {
		// Resolve is a function that will query the manifests service to locate the reference
		Resolve(ctx context.Context, client Client) (descriptor *ocispec.Descriptor, manifest []byte, err error)
		// BlobUploads is a function that will format a URL to the blob uploads service
		BlobUploads() (*url.URL, error)
		// BlobUploads is a function that will format a URL to the blob uploads service with a `mount` and `from` parameter
		BlobUploadMountFrom(mount, from string) (*url.URL, error)
		// Blobs is a function that will format a URL to the blobs service
		Blobs() (*url.URL, error)
		// Manifest is a function that will format a URL to the manifests service
		Manifests() (*url.URL, error)
		// Tags is a function that will format a URL to the tags service
		Tags() (*url.URL, error)
		// TagsFilter is a function that will format a URL to the tags service with filter paramters
		TagsFilter(n, last int) (*url.URL, error)
		// Request is a function that will construct a request object
		Request(ctx context.Context, method string, url *url.URL, body io.Reader) (*http.Request, error)
		// Extension is a function that will construct a request object with the URL provided by the ExtensionAPI function
		Extension(ctx context.Context, method string, body io.Reader, extension ExtendedAPI) (*http.Request, error)
	}
)

type (
	address struct {
		scheme string
		host   string
		ns     string
		loc    string
	}
	reference struct {
		add    address
		media  string
		digest digest.Digest
	}
)

var _ Service = (*reference)(nil)

func (r reference) Resolve(ctx context.Context, client Client) (descriptor *ocispec.Descriptor, manifest []byte, err error) {
	api, err := r.Manifests()
	if err != nil {
		return nil, nil, err
	}

	request, err := r.Request(ctx, http.MethodHead, api, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := client.Do(ctx, request)
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

	desc := &ocispec.Descriptor{
		Digest:    digest.Digest(d),
		MediaType: c,
		Size:      s,
	}

	request, err = r.Request(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err = client.Do(ctx, request)
	if err != nil {
		return nil, nil, err
	}

	mBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	return desc, mBytes, nil
}

func (r reference) BlobUploads() (*url.URL, error) {
	api := fmt.Sprintf("%s/v2/%s/blobs/uploads", r.add.host, r.add.ns)

	if r.add.loc != "" && r.digest != "" {
		api = fmt.Sprintf("%s/%s?digest=%s", api, r.add.loc, r.digest)
	}

	return url.Parse(api)
}

func (r reference) BlobUploadMountFrom(mount, from string) (*url.URL, error) {
	api, err := r.BlobUploads()
	if err != nil {
		return nil, err
	}

	query := api.Query()
	query.Add("mount", mount)
	query.Add("from", from)

	api.RawQuery = query.Encode()

	return api, nil
}

func (r reference) Blobs() (*url.URL, error) {
	api := fmt.Sprintf("%s/v2/%s/blobs/%s", r.add.host, r.add.ns, r.digest)

	return url.Parse(api)
}

func (r reference) Manifests() (*url.URL, error) {
	api := fmt.Sprintf("%s/v2/%s/manifests/%s", r.add.host, r.add.ns, r.add.loc)

	return url.Parse(api)
}

func (r reference) Tags() (*url.URL, error) {
	api := fmt.Sprintf("%s/v2/%s/tags/list/%s", r.add.host, r.add.ns, r.add.loc)

	return url.Parse(api)
}

func (r reference) TagsFilter(n, last int) (*url.URL, error) {
	api, err := r.Tags()
	if err != nil {
		return nil, err
	}

	query := api.Query()
	query.Add("n", fmt.Sprint(n))
	query.Add("last", fmt.Sprint(last))

	api.RawQuery = query.Encode()

	return api, nil
}

func (r reference) Request(ctx context.Context, method string, url *url.URL, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url.String(), body)
	if err != nil {
		return nil, err
	}

	url.Scheme = r.add.scheme

	if body != nil {
		req.Header.Set("Content-Type", r.media)
	} else {
		req.Header.Set("Accept", r.media)
	}

	return req, nil
}

type ExtendedAPI = func(host, namespace, locator, mediaType string, digest digest.Digest) (*url.URL, error)

func (r reference) Extension(ctx context.Context, method string, body io.Reader, extension ExtendedAPI) (*http.Request, error) {
	url, err := extension(r.add.host, r.add.ns, r.add.loc, r.media, r.digest)
	if err != nil {
		return nil, err
	}

	return r.Request(ctx, method, url, body)
}
