package webbrowser

import "strings"

func getEnv(k string, env []string) string {
	for _, e := range env {
		s := strings.Split(e, "=")
		if s[0] == k {
			return s[1]
		}
	}

	return ""
}
