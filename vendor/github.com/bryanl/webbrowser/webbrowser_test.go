package webbrowser

import (
	"reflect"
	"testing"
)

func Test_detectBrowsers(t *testing.T) {
	cases := []struct {
		o string
		t interface{}
		e error
	}{
		{o: "linux", t: &LinuxOpener{}, e: nil},
		{o: "darwin", t: &OSXOpener{}, e: nil},
		{o: "windows", t: &WindowsOpener{}, e: nil},
		{o: "freebsd", e: &UnsupportedOSError{}},
	}

	for _, c := range cases {
		o, err := detectBrowsers(c.o)

		got := reflect.TypeOf(err)
		want := reflect.TypeOf(c.e)
		if got != want {
			t.Fatalf("detectBrowsers() err = %v; want = %v", err, c.e)
		}

		if err == nil {
			got := reflect.TypeOf(o)
			want := reflect.TypeOf(c.t)
			if got != want {
				t.Errorf("detectBrowsers() type = %v; want = %v", got, want)
			}
		}
	}

}
