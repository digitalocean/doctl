/*
Copyright 2018 The Doctl Authors All rights reserved.
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

package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/digitalocean/doctl/internal/ui"
)

func Test_checkErr(t *testing.T) {
	defer func(a func()) { errAction = a }(errAction)

	var b bytes.Buffer
	withUIEnv(t, ui.Plain(&b, &b))

	errAction = func() {
	}

	checkErr(errors.New("an error"))

	// Same label a FlagValidationError carries, so both read as one voice.
	assert.Equal(t, "✗ Error: an error\n", b.String())
}

func Test_checkErr_FlagValidationJSONKeepsErrorsEnvelope(t *testing.T) {
	defer func(a func()) { errAction = a }(errAction)
	defer func() { viper.Set("output", "") }()

	errAction = func() {}
	viper.Set("output", "json")

	// Capture stdout where JSON is printed.
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	checkErr(&FlagValidationError{
		Command: "doctl compute droplet create",
		Issues: []FlagIssue{
			{Flag: "size", Problem: "is required but was not set", Purpose: "Droplet size"},
		},
	})

	require.NoError(t, w.Close())
	os.Stdout = old
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var payload outputErrors
	require.NoError(t, json.Unmarshal(out, &payload))
	require.Len(t, payload.Errors, 1)
	assert.Contains(t, payload.Errors[0].Detail, "--size")
	assert.Contains(t, payload.Errors[0].Detail, "Droplet size")
	assert.NotContains(t, string(out), `"issues"`)
}
