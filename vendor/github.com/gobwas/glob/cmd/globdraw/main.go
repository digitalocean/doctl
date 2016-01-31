package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gobwas/glob"
	"github.com/gobwas/glob/match"
	"math/rand"
	"os"
	"strings"
)

func draw(pattern string, m match.Matcher) string {
	return fmt.Sprintf(`digraph G {graph[label="%s"];%s}`, pattern, graphviz(m, fmt.Sprintf("%x", rand.Int63())))
}

func graphviz(m match.Matcher, id string) string {
	buf := &bytes.Buffer{}

	switch matcher := m.(type) {
	case match.BTree:
		fmt.Fprintf(buf, `"%s"[label="%s"];`, id, matcher.Value.String())
		for _, m := range []match.Matcher{matcher.Left, matcher.Right} {
			switch n := m.(type) {
			case nil:
				rnd := rand.Int63()
				fmt.Fprintf(buf, `"%x"[label="<nil>"];`, rnd)
				fmt.Fprintf(buf, `"%s"->"%x";`, id, rnd)

			default:
				sub := fmt.Sprintf("%x", rand.Int63())
				fmt.Fprintf(buf, `"%s"->"%s";`, id, sub)
				fmt.Fprintf(buf, graphviz(n, sub))
			}
		}

	case match.AnyOf:
		fmt.Fprintf(buf, `"%s"[label="AnyOf"];`, id)
		for _, m := range matcher.Matchers {
			rnd := rand.Int63()
			fmt.Fprintf(buf, graphviz(m, fmt.Sprintf("%x", rnd)))
			fmt.Fprintf(buf, `"%s"->"%x";`, id, rnd)
		}

	case match.EveryOf:
		fmt.Fprintf(buf, `"%s"[label="EveryOf"];`, id)
		for _, m := range matcher.Matchers {
			rnd := rand.Int63()
			fmt.Fprintf(buf, graphviz(m, fmt.Sprintf("%x", rnd)))
			fmt.Fprintf(buf, `"%s"->"%x";`, id, rnd)
		}

	default:
		fmt.Fprintf(buf, `"%s"[label="%s"];`, id, m.String())
	}

	return buf.String()
}

func main() {
	pattern := flag.String("p", "", "pattern to draw")
	sep := flag.String("s", "", "comma separated list of separators")
	flag.Parse()

	if *pattern == "" {
		flag.Usage()
		os.Exit(1)
	}

	glob, err := glob.Compile(*pattern, strings.Split(*sep, ",")...)
	if err != nil {
		fmt.Println("could not compile pattern:", err)
		os.Exit(1)
	}

	matcher := glob.(match.Matcher)
	fmt.Fprint(os.Stdout, draw(*pattern, matcher))
}
