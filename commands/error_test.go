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
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func Test_checkErr(t *testing.T) {
	defer func(a func()) { errAction = a }(errAction)
	defer func(a io.Writer) { color.Output = a }(color.Output)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	color.Output = w

	errAction = func() {
	}

	e := errors.New("an error")
	checkErr(e)
	err := w.Flush()
	assert.NoError(t, err)

	re := regexp.MustCompile(`an error`)
	assert.True(t, re.Match(b.Bytes()))
}
