package urn

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// DefaultNamespace is the default URN namespace for DigitalOcean resources.
	DefaultNamespace = "do"
)

var (
	segmentFormat = `[a-zA-Z0-9-_~\.]{1,256}`
	urnRegexp     = regexp.MustCompile(`^(` + segmentFormat + `):(` + segmentFormat + `):(` + segmentFormat + `)$`)
)

// URN is a unique, uniform identifier for a resource. This structure holds an
// URN parse into its three segments: a namespace, a collection and an identifier.
type URN struct {
	namespace  string
	collection string
	identifier string
}

// ParseURN parses a string representation of an URN and returns its structured representation (*URN).
func ParseURN(s string) (*URN, error) {
	segments := urnRegexp.FindStringSubmatch(s)
	if len(segments) < 4 {
		return nil, errors.New("invalid urn")
	}

	return NewURN(segments[1], segments[2], segments[3]), nil
}

// NewURN constructs an *URN from a namespace, resource type, and identifier.
func NewURN(namespace string, collection string, id interface{}) *URN {
	return &URN{
		namespace:  strings.ToLower(namespace),
		collection: strings.ToLower(collection),
		identifier: fmt.Sprintf("%v", id),
	}
}

// Namespace returns the namespace segment of the URN.
func (u *URN) Namespace() string {
	return u.namespace
}

// Collection returns the collection segment of the URN.
func (u *URN) Collection() string {
	return u.collection
}

// Identifier returns the identifier segment of the URN.
func (u *URN) Identifier() string {
	return u.identifier
}

// String returns a string representation of the URN.
func (u *URN) String() string {
	return fmt.Sprintf("%s:%s:%s", u.namespace, u.collection, u.identifier)
}
