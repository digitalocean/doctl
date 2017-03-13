// +build !linux,!darwin

package errors

import (
	"io"
	"strings"
)

// Test if err indicates EOF
func EOF(err error) bool {
	if err == io.EOF {
		return true
	}

	if strings.Contains(err.Error(), "closed") {
		return true
	}

	if strings.Contains(err.Error(), "reset by peer") {
		return true
	}

	return false
}
