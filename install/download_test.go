package install

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	appFS := &afero.MemMapFs{}

	appFS.Mkdir("test", 0755)
	afero.WriteFile(appFS, "test/dl", []byte("dl"), 0644)
	afero.WriteFile(appFS, "test/dl.sha256", []byte("2ca69efd4ea5af91a637f19ba0bab8b081d2c03773c4a72fcbf8817c856b33ef  /test/dl.sha256"), 0644)

	dl, err := appFS.Open("test/dl")
	assert.NoError(t, err)

	cs, err := appFS.Open("test/dl.sha256")
	assert.NoError(t, err)

	err = Validate(dl, cs)
	assert.NoError(t, err)
}
