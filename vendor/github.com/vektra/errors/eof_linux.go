package errors

import (
	"io"
	"syscall"
)

// Test if err indicates EOF
func EOF(err error) bool {
	if err == io.EOF {
		return true
	}

	if serr, ok := err.(syscall.Errno); ok {
		switch serr {
		case syscall.ECONNREFUSED:
			return true
		case syscall.ECONNRESET:
			return true
		case syscall.ENOTCONN:
			return true
		case syscall.ENETDOWN:
			return true
		case syscall.ENETUNREACH:
			return true
		case syscall.ETIMEDOUT:
			return true
		default:
			return false
		}
	}

	return false
}
