package workspace

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ensureStringInFile(t *testing.T) {
	ensureValue := "newvalue"

	tcs := []struct {
		name   string
		pre    func(t *testing.T, fname string)
		expect []byte
	}{
		{
			name:   "no pre-existing file",
			pre:    func(t *testing.T, fname string) {},
			expect: []byte(ensureValue),
		},
		{
			name: "pre-existing file with value",
			pre: func(t *testing.T, fname string) {
				f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				require.NoError(t, err)
				f.WriteString("line1\n" + ensureValue)
				f.Close()
			},
			expect: []byte("line1\n" + ensureValue),
		},
		{
			name: "pre-existing file without value",
			pre: func(t *testing.T, fname string) {
				f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				require.NoError(t, err)
				f.WriteString("line1\n")
				f.Close()
			},
			expect: []byte("line1\n" + ensureValue),
		},
		{
			name: "pre-existing file without value or newline",
			pre: func(t *testing.T, fname string) {
				f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				require.NoError(t, err)
				f.WriteString("line1")
				f.Close()
			},
			expect: []byte("line1\n" + ensureValue),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			file, err := ioutil.TempFile("", "dev-config.*.yaml")
			require.NoError(t, err, "creating temp file")
			file.Close()

			// allow the test to dictate existence; we just use this
			// to get a valid temporary filename that is unique
			err = os.Remove(file.Name())
			require.NoError(t, err, "deleting temp file")

			if tc.pre != nil {
				tc.pre(t, file.Name())
			}

			err = ensureStringInFile(file.Name(), ensureValue)
			require.NoError(t, err, "ensuring string in file")

			b, err := ioutil.ReadFile(file.Name())
			require.NoError(t, err)
			require.Equal(t, string(tc.expect), string(b))
		})
	}
}
