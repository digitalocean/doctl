package urn

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURN(t *testing.T) {

	tests := []struct {
		name        string
		in          string
		expected    *URN
		expectedErr error
	}{
		{
			name: "urn with int id",
			in:   "do:droplet:123456",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "droplet",
				identifier: "123456",
			},
		},
		{
			name: "urn with string id",
			in:   "do:kubernetes:ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "kubernetes",
				identifier: "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			},
		},
		{
			name: "urn with capitalized input",
			in:   "DO:Kubernetes:ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "kubernetes",
				identifier: "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			},
		},
		{
			name:        "invalid urn with too many segments",
			in:          "DO:Kubernetes:123:abc",
			expectedErr: errors.New("invalid urn"),
		},
		{
			name:        "not an urn",
			in:          "9adfd91e-a6f8-11ec-970d-dfc761e0603d",
			expectedErr: errors.New("invalid urn"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURN(tt.in)
			if tt.expectedErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, got)

		})
	}
}

func TestNewURN(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		collection string
		identifier interface{}
		expected   *URN
		asString   string
	}{
		{
			name:       "urn with int id",
			namespace:  DefaultNamespace,
			collection: "droplet",
			identifier: 123456,
			asString:   "do:droplet:123456",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "droplet",
				identifier: "123456",
			},
		},
		{
			name:       "urn with string id",
			namespace:  DefaultNamespace,
			collection: "droplet",
			identifier: 123456,
			asString:   "do:droplet:123456",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "droplet",
				identifier: "123456",
			},
		},
		{
			name:       "urn with string uuid",
			namespace:  DefaultNamespace,
			collection: "kubernetes",
			identifier: "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			asString:   "do:kubernetes:ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "kubernetes",
				identifier: "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			},
		},
		{
			name:       "urn with capitalized input",
			namespace:  DefaultNamespace,
			collection: "Kubernetes",
			identifier: "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			asString:   "do:kubernetes:ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			expected: &URN{
				namespace:  DefaultNamespace,
				collection: "kubernetes",
				identifier: "ca0cc702-a6f7-11ec-a8fc-6bcc60f7e984",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewURN(tt.namespace, tt.collection, tt.identifier)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.namespace, got.Namespace())
			assert.Equal(t, strings.ToLower(tt.collection), got.Collection())
			assert.Equal(t, strings.ToLower(fmt.Sprintf("%v", tt.identifier)), got.Identifier())
			assert.Equal(t, tt.asString, got.String())
		})
	}
}
