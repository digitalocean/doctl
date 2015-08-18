package doit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigFromBytes(t *testing.T) {
	cfg :=
		`
global_opt: 1
top1:
  another_opt: string
  nested2:
    bool_opt: true
`

	config, err := NewConfig2([]byte(cfg))
	assert.NoError(t, err)

	i, err := config.Int("global_opt")
	assert.NoError(t, err)
	assert.Equal(t, 1, i)

	s, err := config.String("top1.another_opt")
	assert.NoError(t, err)
	assert.Equal(t, "string", s)

	b, err := config.Bool("top1.nested2.bool_opt")
	assert.NoError(t, err)
	assert.True(t, b)
}
