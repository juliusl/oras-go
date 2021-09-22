package remotes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"oras.land/oras-go/pkg/content"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func NewRegistry(host, ns string, provider AccessProvider) *Registry {
	return &Registry{
		provider:    provider,
		host:        host,
		namespace:   ns,
		descriptors: make(map[reference]*ocispec.Descriptor),
		manifest:    make(map[reference]*ocispec.Manifest),
	}
}

type (
	// Doer is an interface for doing
	Doer interface {
		// Do is a function that sends the request and handles the response
		Do(ctx context.Context, req *http.Request) (*http.Response, error)
	}

	// Registry is an opaqueish type which represents an OCI V2 API registry
	Registry struct {
		host        string
		namespace   string
		provider    AccessProvider
		descriptors map[reference]*ocispec.Descriptor
		manifest    map[reference]*ocispec.Manifest
		*http.Client
	}
)

// Do is a function that does the request, error handling is concentrated in this method
func (r *Registry) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if r.Client == nil {
		r.setClient(http.DefaultClient) // TODO make a default anonymous client (tune to fail fast since most things need to be authenticated)
	}

	resp, err := r.do(req)
	if err != nil {
		// This comes from the redirect handler
		ne, ok := err.(*url.Error)
		if ok {
			re, ok := ne.Err.(*RedirectError)
			if ok {
				resp, err = re.Retry(r.Client)
				if err != nil {
					resp.Body.Close()
					return nil, err
				}

				return resp, nil
			}
		}

		if errors.Is(err, AuthChallengeErr) {
			challengeError, ok := err.(*AuthChallengeError)
			if ok {
				// Check our provider for access
				access, err := r.provider.GetAccess(ctx, challengeError)
				if err != nil {
					if resp != nil && resp.Request != nil {
						defer resp.Body.Close()
					}

					return nil, err
				}

				// Get a new client once we have access
				c, err := access.GetClient(ctx)
				if err != nil {
					if resp != nil && resp.Request != nil {
						defer resp.Body.Close()
					}
					return nil, err
				}
				r.setClient(c)

				resp, err = c.Do(req)
				if err != nil {
					return nil, err
				}

				return resp, nil
			}
		}

		return nil, err
	}

	return resp, nil
}

func (r *Registry) setClient(client *http.Client) {
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && req.URL.Host != via[0].Host &&
			req.Header.Get("Authorization") == via[0].Header.Get("Authorization") {
			return NewRedirectError(req)
		}
		return nil
	}
	r.Client = client
}

// do calls the concrete http client, and handles http related status code issues
func (r *Registry) do(req *http.Request) (*http.Response, error) {
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			c, ok := resp.Header["Www-Authenticate"]
			if ok {
				// TODO not sure what the delimitter would be
				return nil, NewAuthChallengeError(strings.Join(c, ","))
			}

			return nil, fmt.Errorf("not authenticated")
		}

		defer resp.Body.Close()

		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

type CONTENT_WRITER_ALIAS = *content.OCIStore

type RegistryFunctions struct {
	fetcher    func(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error)
	pusher     func(ctx context.Context, desc ocispec.Descriptor) (CONTENT_WRITER_ALIAS, error)
	resolver   func(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error)
	discoverer func(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error)
}

func (r *Registry) AsFunctions() *RegistryFunctions {
	return &RegistryFunctions{
		fetcher:    r.fetch,
		pusher:     r.push,
		resolver:   r.resolve,
		discoverer: r.discover,
	}
}

func (r *RegistryFunctions) Pusher() func(ctx context.Context, desc ocispec.Descriptor) (CONTENT_WRITER_ALIAS, error) {
	return r.pusher
}

func (r *RegistryFunctions) Fetcher() func(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	return r.fetcher
}

func (r *RegistryFunctions) Resolver() func(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	return r.resolver
}

func (r *RegistryFunctions) Discoverer() func(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error) {
	return r.discoverer
}

type address struct {
	host string
	ns   string
	loc  string
}

type reference struct {
	add   address
	media string
	digst digest.Digest
}

func (r *Registry) GetManifest(ctx context.Context, ref string) (*ocispec.Descriptor, *ocispec.Manifest, error) {
	host, ns, loc, err := Parse(ref)
	if err != nil {
		return nil, nil, fmt.Errorf("reference is is not valid")
	}

	if ns != r.namespace {
		return nil, nil, fmt.Errorf("namespace does not match current registry context")
	}

	if host != r.host {
		return nil, nil, fmt.Errorf("host does not match current registry context")
	}

	// format the reference
	manifestRef := reference{
		add: address{
			host: r.host,
			ns:   r.namespace,
			loc:  loc,
		},
		digst: "",
	}

	m := manifests{ref: manifestRef}

	desc, spec, err := m.getManifest(ctx, r)
	if err != nil {
		return nil, nil, err
	}

	if spec.Annotations == nil {
		spec.Annotations = make(map[string]string)
	}

	spec.Annotations["host"] = host
	spec.Annotations["namespace"] = ns
	spec.Annotations["loc"] = loc

	return desc, spec, nil
}

// resolve resolves a reference to a descriptor
func (r *Registry) resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	d, _, err := r.GetManifest(ctx, ref)
	if err != nil {
		return "", ocispec.Descriptor{}, err
	}

	return ref, *d, nil
}

func (r *Registry) fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	if r == nil {
		return nil, fmt.Errorf("reference is nil")
	}

	b := blob{
		ref: reference{
			add: address{
				host: r.host,
				ns:   r.namespace,
				loc:  "",
			},
			digst: desc.Digest,
			media: desc.MediaType,
		},
	}

	return b.fetch(ctx, r)
}

func (r *Registry) discover(ctx context.Context, desc ocispec.Descriptor, artifactType string) (*Artifacts, error) {
	if r == nil {
		return nil, fmt.Errorf("reference is nil")
	}

	return artifacts{
		artifactType: artifactType,
		ref: reference{
			add: address{
				host: r.host,
				ns:   r.namespace,
				loc:  "",
			},
			digst: desc.Digest,
			media: desc.MediaType,
		},
	}.discover(ctx, r)
}

func (r *Registry) push(ctx context.Context, desc ocispec.Descriptor) (CONTENT_WRITER_ALIAS, error) {

	return nil, fmt.Errorf("push api has not been implemented") // TODO
}
