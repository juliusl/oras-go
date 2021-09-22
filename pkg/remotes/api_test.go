package remotes

import (
	"fmt"
	"testing"

	"github.com/opencontainers/go-digest"
)

func TestReferences(t *testing.T) {
	testRef{
		host: "localhost:5000",
		ns:   "oras",
		loc:  "v1",
	}.Test(t)

	testRef{
		host: "registry-1.docker.io",
		ns:   "library/ubuntu",
		loc:  "latest",
	}.Test(t)

	testRef{
		host: "ghcr.io",
		ns:   "oras-project/registry",
		loc:  "v0.0.3-alpha",
	}.Test(t)

	testRef{
		host:   "ghcr.io",
		ns:     "oras-project/registry",
		digest: digest.FromString("sha256:4942a1abcbfa1c325b1d7ed93d3cf6020f555be706672308a4a4a6b6d631d2e7"),
	}.Test(t)
}

type testRef struct {
	host, ns, loc string
	digest        digest.Digest
}

func (r testRef) GetRef() string {
	if r.digest != "" {
		return fmt.Sprintf("%s/%s@%s", r.host, r.ns, r.digest)
	} else {
		return fmt.Sprintf("%s/%s:%s", r.host, r.ns, r.loc)
	}
}

func (r testRef) Test(t *testing.T) {
	_, host, ns, loc, err := Parse(r.GetRef())
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	if host != r.host {
		t.Fail()
	}

	if ns != r.ns {
		t.Fail()
	}

	if r.digest != "" {
		r.loc = r.digest.String()
	}

	if loc != r.loc {
		t.Fail()
	}
}
