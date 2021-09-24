package remotes

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Table of endpoints for OCI v2
// end-1	GET			/v2/														200	404/401
// end-2	GET / HEAD	/v2/<name>/blobs/<digest>									200	404
// end-10	DELETE		/v2/<name>/blobs/<digest>									202	404/405
// end-4a	POST		/v2/<name>/blobs/uploads/									202	404
// end-4b	POST		/v2/<name>/blobs/uploads/?digest=<digest>					201/202	404/400
// end-5	PATCH		/v2/<name>/blobs/uploads/<reference>						202	404/416
// end-6	PUT			/v2/<name>/blobs/uploads/<reference>?digest=<digest>		201	404/400
// end-11	POST		/v2/<name>/blobs/uploads/?mount=<digest>&from=<other_name>	201	404

// end-3	GET / HEAD	/v2/<name>/manifests/<reference>							200	404
// end-7	PUT			/v2/<name>/manifests/<reference>							201	404
// end-9	DELETE		/v2/<name>/manifests/<reference>							202	404/400/405
// end-8a	GET			/v2/<name>/tags/list										200	404
// end-8b	GET			/v2/<name>/tags/list?n=<integer>&last=<integer>				200	404

// ORAS
// get-signatures	GET		/oras/artifacts/v1/<name>/manifests/<digest>											200 404/401
// list-referrers	GET		/oras/artifacts/v1/<name>/manifests/<digest>/referrers?artifactType=<artifacttype>		200 404/401

// 	# Value conformance
// <name>		   - is the namespace of the repository, must match [a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*
// <reference>     - is either a digest or a tag, must match [a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}
// <artifacttype>  - analagous to reference except that it allows for symbols

var (
	referenceRegex = regexp.MustCompile(`([.\w\d:-]+)\/{1,}?([a-z0-9]+(?:[/._-][a-z0-9]+)*(?:[a-z0-9]+(?:[/._-][a-z0-9]+)*)*)[:@]([a-zA-Z0-9_]+:?[a-zA-Z0-9._-]{0,127})`)
)

func Parse(parsing string) (reference string, host string, namespace string, locator string, err error) {
	matches := referenceRegex.FindAllStringSubmatch(parsing, -1)
	// Technically a namespace is allowed to have "/"'s, while a reference is not allowed to
	// That means if you string match the reference regex, then you should end up with basically the first segment being the host
	// the middle part being the namespace
	// and the last part should be the tag

	// This should be the case most of the time
	if len(matches[0]) == 4 {
		return matches[0][0], matches[0][1], matches[0][2], matches[0][3], nil
	}

	return "", "", "", "", errors.New("could not parse reference")
}

func ValidateReference(reference string) (string, error) {
	matches := referenceRegex.FindAllString(reference, -1)

	if len(matches) <= 0 {
		return "", fmt.Errorf("either the reference was empty, or it contained no characters")
	}

	maybe := matches[len(matches)-1]

	endsWith := strings.HasSuffix(reference, ":"+maybe) || strings.HasPrefix(reference, "@"+maybe)
	if endsWith {
		return maybe, nil
	}

	return "", fmt.Errorf("malformed reference, a reference should be in the form of {host}/{namespace}:{tag}")
}

// code-1	BLOB_UNKNOWN			blob unknown to registry
// code-2	BLOB_UPLOAD_INVALID		blob upload invalid
// code-3	BLOB_UPLOAD_UNKNOWN		blob upload unknown to registry
// code-4	DIGEST_INVALID			provided digest did not match uploaded content
// code-5	MANIFEST_BLOB_UNKNOWN	blob unknown to registry
// code-6	MANIFEST_INVALID		manifest invalid
// code-7	MANIFEST_UNKNOWN		manifest unknown
// code-8	NAME_INVALID			invalid repository name
// code-9	NAME_UNKNOWN			repository name not known to registry
// code-10	SIZE_INVALID			provided length did not match content length
// code-12	UNAUTHORIZED			authentication required
// code-13	DENIED					requested access to the resource is denied
// code-14	UNSUPPORTED				the operation is unsupported
// code-15	TOOMANYREQUESTS			too many requests
