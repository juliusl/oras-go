/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package target

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

// Object is an opaque type that implements functions to locate and reference objects
type Object struct {
	reference    string
	artifactType string
	digest       digest.Digest
	mediaType    string
	size         int64
	annotations  map[string]string
	subject      *Object
}

// ReferenceSpec is a function that returns the reference specification for this object
func (o Object) ReferenceSpec() (reference string, host string, namespace string, object string, err error) {
	return parse(o.reference)
}

// IsRoot is a function that returns whether the current object is a root object
func (o Object) IsRoot() bool {
	return o.subject == nil
}

// Descriptor is a function that returns the ocispec descriptor struct that can represent this object
func (o Object) Descriptor() ocispec.Descriptor {
	return ocispec.Descriptor{
		Digest:      o.digest,
		Size:        o.size,
		MediaType:   o.mediaType,
		Annotations: o.annotations,
	}
}

// ArtifactDescriptor is a function that returns the artifact spec descriptor view of this object
func (o Object) ArtifactDescriptor() artifactspec.Descriptor {
	return artifactspec.Descriptor{
		Digest:       o.digest,
		Size:         o.size,
		MediaType:    o.mediaType,
		Annotations:  o.annotations,
		ArtifactType: o.artifactType,
	}
}

// Manifest is a function that returns the Manifest view of this object
func (o Object) ArtifactManifest() artifactspec.Manifest {
	subject := o.subject.ArtifactDescriptor()

	return artifactspec.Manifest{
		ArtifactType: o.artifactType,
		Subject:      subject,
		Blobs:        []artifactspec.Descriptor{o.ArtifactDescriptor()},
	}
}

// ResolveSubject is a function that resolves the subject of this object located in a target
func (o Object) ResolveSubject(ctx context.Context, target Target, withRef string) (Object, error) {
	if o.IsRoot() {
		return Object{}, errors.New("current object is a root object")
	}

	if withRef == "" {
		withRef = o.subject.reference
	}

	_, desc, err := target.Resolve(ctx, withRef)
	if err != nil {
		return Object{}, err
	}

	return FromOCIDescriptor(o.subject.reference, desc, o.subject.artifactType, nil), nil
}

// Subject is a function that returns this object's subject
func (o Object) Subject() *Object {
	return o.subject
}

// List is a function that writes records associated to this object, starting from the root
// The fields that are written are mediatype and reference
func (o Object) List(fieldSeperator []byte, writer io.Writer) {
	if o.subject != nil {
		o.subject.List(fieldSeperator, writer)
	}

	writer.Write([]byte(fmt.Sprint(o.mediaType)))
	writer.Write(fieldSeperator)
	writer.Write([]byte(fmt.Sprint(o.reference)))
	writer.Write([]byte("\n"))
}

// Move is a function that moves this object from an artifact to a target
func (o Object) Move(ctx context.Context, from Artifact, to Target, toLocator string) error {
	// toLocator is a reference spec, parse the host and namespace portion to construct the new reference spec
	_, host, namespace, _, err := parse(toLocator)
	if err != nil {
		return err
	}

	_, _, _, object, err := parse(o.reference)
	if err != nil {
		return err
	}

	if o.subject != nil {
		defer o.subject.Move(ctx, from, to, toLocator)
	}

	desc := o.Descriptor()

	reader, err := from.ReaderAt(ctx, desc)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Optionally the locator can omit the namespace portion and include just a new host
	// the current namespace will be reused
	if namespace == "" {
		_, _, namespace, _, err = parse(o.reference)
		if err != nil {
			return err
		}
	}

	ref := fmt.Sprintf("%s/%s%s", host, namespace, object)

	pusher, err := to.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	writer, err := pusher.Push(ctx, desc)
	if err != nil {
		return err
	}

	err = content.Copy(ctx, writer, content.NewReader(reader), desc.Size, desc.Digest)
	if err != nil {
		return err
	}

	fmt.Printf("Moved %s %s to %s\n", o.mediaType, o.reference, ref)

	return nil
}

// Download is a function that downloads this object from a target to an artifact
func (o Object) Download(ctx context.Context, from Target, to Artifact) error {
	desc := o.Descriptor()

	fetcher, err := from.Fetcher(ctx, o.reference)
	if err != nil {
		return err
	}

	writer, err := to.Writer(ctx, content.WithDescriptor(desc), content.WithRef(o.reference))
	if err != nil {
		return err
	}
	defer writer.Close()

	reader, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = content.Copy(ctx, writer, reader, desc.Size, desc.Digest)
	if err != nil {
		return err
	}

	return nil
}

// MarshalJson is a function that deserializes the content of this object as json
func (o Object) MarshalJson(ctx context.Context, source Target, out interface{}) error {
	err := o.MarshalObject(ctx, source, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(out)
	})
	if err != nil {
		return err
	}

	return nil
}

// MarshalObject is a function that provides a general purpose marshalling entrypoint
func (o Object) MarshalObject(ctx context.Context, source Target, marshaller func(io.Reader) error) error {
	fetcher, err := source.Fetcher(ctx, o.reference)
	if err != nil {
		return err
	}

	reader, err := fetcher.Fetch(ctx, o.Descriptor())
	if err != nil {
		return err
	}
	defer reader.Close()

	err = marshaller(reader)
	if err != nil {
		return err
	}

	return nil
}

// MarshalJsonFromArtifact is a function that marshals from an artifact as the source instead of a target
func (o Object) MarshalJsonFromArtifact(ctx context.Context, source Artifact, out interface{}) error {
	err := o.MarshalObjectFromArtifact(ctx, source, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(out)
	})
	if err != nil {
		return err
	}

	return nil
}

// MarshalObjectFromArtifact is a function that marshals an object from an artifact instead of a target
func (o Object) MarshalObjectFromArtifact(ctx context.Context, source Artifact, marshaller func(io.Reader) error) error {
	reader, err := source.ReaderAt(ctx, o.Descriptor())
	if err != nil {
		return err
	}
	defer reader.Close()

	err = marshaller(content.NewReader(reader))
	if err != nil {
		return err
	}

	return nil
}

var (
	// referenceRegex is a regular expression that parses and returns parts of a reference specification
	// originally the reference specification is broken down into 2 parts, the Locator and Object
	// The first part, Locator is the host and namespace of the reference specification
	// The second part Object is either a tag or digest
	//
	// This expression might look complicated, but it is quite straight-forward. The length of the expression is due to the fact it does not use any lookaround methods,
	// which means it must declare explicitly the path it takes to parse for matches, hence the straight-forwardness.
	// This expression is composed of 3 capture groups and returns 4 values. The first value is the reference that is being parsed,
	// there is not a specific reason for this, but traditionally when starting a process argv[0] is generally the name of the program, and in awk $0 is the entire line
	// The first and second capture group find the two main parts of the Locator which are the Hostname (shorted as host) and namespace
	// the final capture group searches for the object, and does not make a distinction between the tag or digest format, but instead limits the total length value to 127 characters.
	// This last capture group uses the regular expression defined in the oci-distribution specification
	referenceRegex = regexp.MustCompile(`([.\w\d:-]+)\/{1,}?([a-z0-9]+(?:[/._-][a-z0-9]+)*(?:[a-z0-9]+(?:[/._-][a-z0-9]+)*)*)([:@][a-zA-Z0-9_]+:?[a-zA-Z0-9._-]{0,127})`)
)

func parse(parsing string) (reference string, host string, namespace string, object string, err error) {
	matches := referenceRegex.FindAllStringSubmatch(parsing, -1)
	// Technically a namespace is allowed to have "/"'s, while a reference is not allowed to
	// That means if you string match the reference regex, then you should end up with basically the first segment being the host
	// the middle part being the namespace
	// and the last part should be the tag

	// This should be the case most of the time
	if len(matches) > 0 && len(matches[0]) == 4 {
		return matches[0][0], matches[0][1], matches[0][2], matches[0][3], nil
	}

	return "", "", "", "", errors.New("could not parse reference")
}
