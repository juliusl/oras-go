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
				resp, err := re.Retry(r.Client)
				if err != nil {
					resp.Body.Close()
					return nil, err
				}

				return resp, nil
			}
		}

		if errors.Is(err, RedirectError{}) {
			redirectErr, ok := errors.Unwrap(err).(*RedirectError)
			if ok {
				// Can't use the built in client, because it will add the Authorization header
				// TODO - but still shouldn't use DefaultClient
				resp, err = redirectErr.Retry(http.DefaultClient)
				if err != nil {
					resp.Body.Close()
					return nil, err
				}

				return resp, nil
			} else {
				resp.Body.Close()
				return nil, err
			}
		}

		if errors.Is(err, AuthChallengeErr) {
			challengeError, ok := err.(*AuthChallengeError)
			if ok {
				// Check our provider for access
				access, err := r.provider.GetAccess(ctx, challengeError)
				if err != nil {
					resp.Body.Close()
					return nil, err
				}

				// Get a new client once we have access
				c, err := access.GetClient(ctx)
				if err != nil {
					resp.Body.Close()
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
			req.Header.Del("Authorization")
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
		}

		defer resp.Body.Close()

		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

type RegistryFunctions struct {
	fetcher    func(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error)
	pusher     func(ctx context.Context, desc ocispec.Descriptor) (*content.PassthroughWriter, error)
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

func (r *RegistryFunctions) Pusher() func(ctx context.Context, desc ocispec.Descriptor) (*content.PassthroughWriter, error) {
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

// ping ensures that the registry is alive and a registry
func (r *Registry) ping(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("reference is nil")
	}

	request, err := endpoints.e1.prepare()(ctx, r.host, r.namespace, "")
	if err != nil {
		return err
	}

	resp, err := r.Do(ctx, request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("non successful error code %d", resp.StatusCode)
	}

	return nil
}

// resolve resolves a reference to a descriptor
func (r *Registry) resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	if r == nil {
		return "", ocispec.Descriptor{}, fmt.Errorf("registry is nil")
	}

	// // ensure the registry is running
	// err = r.ping(ctx)
	// if err != nil {
	// 	return "", ocispec.Descriptor{}, err
	// }

	host, ns, loc, err := Parse(ref)
	if err != nil {
		return "", ocispec.Descriptor{}, fmt.Errorf("reference is is not valid")
	}

	if ns != r.namespace {
		return "", ocispec.Descriptor{}, fmt.Errorf("namespace does not match current registry context")
	}

	if host != r.host {
		return "", ocispec.Descriptor{}, fmt.Errorf("host does not match current registry context")
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

	// format the manifests request
	m := manifests{ref: manifestRef}

	// Return early if we can get the manifest early
	desc, err = m.getDescriptor(ctx, r)
	// if err == nil && desc.Digest != "" {
	// 	// manifestRef.digst = desc.Digest
	// 	// manifestRef.media = desc.MediaType
	// 	// r.descriptors[manifestRef] = &desc

	// 	// return ref, desc, nil
	// }

	// Get the manifest to retrieve the desc
	manifest, err := m.getDescriptorWithManifest(ctx, r)
	if err != nil {
		return "", ocispec.Descriptor{}, err
	}

	manifestRef.digst = desc.Digest
	r.descriptors[manifestRef] = &manifest.Config
	r.manifest[manifestRef] = manifest

	return ref, manifest.Config, nil
}

func (r *Registry) fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	if r == nil {
		return nil, fmt.Errorf("reference is nil")
	}

	// ensure the registry is running
	// err := r.ping(ctx)
	// if err != nil {
	// 	return nil, err
	// }

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

	// ensure the registry is running
	// err := r.ping(ctx)
	// if err != nil {
	// 	return nil, err
	// }

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

func (r *Registry) push(ctx context.Context, desc ocispec.Descriptor) (*content.PassthroughWriter, error) {

	return &content.PassthroughWriter{}, fmt.Errorf("push api has not been implemented") // TODO
}
