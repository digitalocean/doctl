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
	"fmt"
	"strconv"
)

// ContextualAtoi cleans the error output of Atoi calls
func ContextualAtoi(s, resource string) (int, error) {
	n, err := strconv.Atoi(s)
	if err == nil {
		if n < 0 {
			return 0, fmt.Errorf("expected %d to be a positive integer", n)
		}
		return n, nil
	}
	if _, ok := err.(*strconv.NumError); ok {
		return 0, fmt.Errorf(`expected %s to be a positive integer, got "%s"`, resource, s)
	}
	return 0, err
}
