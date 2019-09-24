package displayers

import (
	"bytes"
	"testing"

	"github.com/digitalocean/doctl/do"

	"github.com/stretchr/testify/assert"
)

func TestDisplayerDisplay(t *testing.T) {
	emptyVolumes := make([]do.Volume, 0)
	var nilVolumes []do.Volume

	tests := []struct {
		name         string
		item         Displayable
		expectedJSON string
	}{
		{
			name:         "displaying a non-nil slice of Volumes should return an empty JSON array",
			item:         &Volume{Volumes: emptyVolumes},
			expectedJSON: `[]`,
		},
		{
			name:         "displaying a nil slice of Volumes should return an empty JSON array",
			item:         &Volume{Volumes: nilVolumes},
			expectedJSON: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}

			displayer := Displayer{
				OutputType: "json",
				Item:       tt.item,
				Out:        out,
			}

			err := displayer.Display()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedJSON, out.String())
		})
	}
}
