package docli

import (
	"os"
	"testing"
)

var lastBailOut bailOut

type bailOut struct {
	err error
	msg string
}

func TestMain(m *testing.M) {
	Bail = func(err error, msg string) {
		lastBailOut = bailOut{
			err: err,
			msg: msg,
		}
	}

	os.Exit(m.Run())
}
