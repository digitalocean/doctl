package glob

import (
	"github.com/gobwas/glob/match"
	"reflect"
	"testing"
)

const separators = "."

func TestGlueMatchers(t *testing.T) {
	for id, test := range []struct {
		in  []match.Matcher
		exp match.Matcher
	}{
		{
			[]match.Matcher{
				match.Super{},
				match.Single{},
			},
			match.Min{1},
		},
		{
			[]match.Matcher{
				match.Any{separators},
				match.Single{separators},
			},
			match.EveryOf{match.Matchers{
				match.Min{1},
				match.Contains{separators, true},
			}},
		},
		{
			[]match.Matcher{
				match.Single{},
				match.Single{},
				match.Single{},
			},
			match.EveryOf{match.Matchers{
				match.Min{3},
				match.Max{3},
			}},
		},
		{
			[]match.Matcher{
				match.List{"a", true},
				match.Any{"a"},
			},
			match.EveryOf{match.Matchers{
				match.Min{1},
				match.Contains{"a", true},
			}},
		},
	} {
		act, err := compileMatchers(test.in)
		if err != nil {
			t.Errorf("#%d convert matchers error: %s", id, err)
			continue
		}

		if !reflect.DeepEqual(act, test.exp) {
			t.Errorf("#%d unexpected convert matchers result:\nact: %s;\nexp: %s", id, act, test.exp)
			continue
		}
	}
}

func TestCompileMatchers(t *testing.T) {
	for id, test := range []struct {
		in  []match.Matcher
		exp match.Matcher
	}{
		{
			[]match.Matcher{
				match.Super{},
				match.Single{separators},
				match.NewText("c"),
			},
			match.NewBTree(
				match.NewText("c"),
				match.NewBTree(
					match.Single{separators},
					match.Super{},
					nil,
				),
				nil,
			),
		},
		{
			[]match.Matcher{
				match.Any{},
				match.NewText("c"),
				match.Any{},
			},
			match.NewBTree(
				match.NewText("c"),
				match.Any{},
				match.Any{},
			),
		},
		{
			[]match.Matcher{
				match.Range{'a', 'c', true},
				match.List{"zte", false},
				match.NewText("c"),
				match.Single{},
			},
			match.Row{
				Matchers: match.Matchers{
					match.Range{'a', 'c', true},
					match.List{"zte", false},
					match.NewText("c"),
					match.Single{},
				},
				RunesLength: 4,
			},
		},
	} {
		act, err := compileMatchers(test.in)
		if err != nil {
			t.Errorf("#%d convert matchers error: %s", id, err)
			continue
		}

		if !reflect.DeepEqual(act, test.exp) {
			t.Errorf("#%d unexpected convert matchers result:\nact: %s;\nexp: %s", id, act, test.exp)
			continue
		}
	}
}

func TestConvertMatchers(t *testing.T) {
	for id, test := range []struct {
		in, exp []match.Matcher
	}{
		{
			[]match.Matcher{
				match.Range{'a', 'c', true},
				match.List{"zte", false},
				match.NewText("c"),
				match.Single{},
				match.Any{},
			},
			[]match.Matcher{
				match.Row{
					Matchers: match.Matchers{
						match.Range{'a', 'c', true},
						match.List{"zte", false},
						match.NewText("c"),
						match.Single{},
					},
					RunesLength: 4,
				},
				match.Any{},
			},
		},
		{
			[]match.Matcher{
				match.Range{'a', 'c', true},
				match.List{"zte", false},
				match.NewText("c"),
				match.Single{},
				match.Any{},
				match.Single{},
				match.Single{},
				match.Any{},
			},
			[]match.Matcher{
				match.Row{
					Matchers: match.Matchers{
						match.Range{'a', 'c', true},
						match.List{"zte", false},
						match.NewText("c"),
					},
					RunesLength: 3,
				},
				match.Min{3},
			},
		},
	} {
		act := minimizeMatchers(test.in)
		if !reflect.DeepEqual(act, test.exp) {
			t.Errorf("#%d unexpected convert matchers 2 result:\nact: %s;\nexp: %s", id, act, test.exp)
			continue
		}
	}
}

func pattern(nodes ...node) *nodePattern {
	return &nodePattern{
		nodeImpl: nodeImpl{
			desc: nodes,
		},
	}
}
func anyOf(nodes ...node) *nodeAnyOf {
	return &nodeAnyOf{
		nodeImpl: nodeImpl{
			desc: nodes,
		},
	}
}
func TestCompiler(t *testing.T) {
	for id, test := range []struct {
		ast    *nodePattern
		result Glob
		sep    string
	}{
		{
			ast:    pattern(&nodeText{text: "abc"}),
			result: match.NewText("abc"),
		},
		{
			ast:    pattern(&nodeAny{}),
			sep:    separators,
			result: match.Any{separators},
		},
		{
			ast:    pattern(&nodeAny{}),
			result: match.Super{},
		},
		{
			ast:    pattern(&nodeSuper{}),
			result: match.Super{},
		},
		{
			ast:    pattern(&nodeSingle{}),
			sep:    separators,
			result: match.Single{separators},
		},
		{
			ast: pattern(&nodeRange{
				lo:  'a',
				hi:  'z',
				not: true,
			}),
			result: match.Range{'a', 'z', true},
		},
		{
			ast: pattern(&nodeList{
				chars: "abc",
				not:   true,
			}),
			result: match.List{"abc", true},
		},
		{
			ast: pattern(&nodeAny{}, &nodeSingle{}, &nodeSingle{}, &nodeSingle{}),
			sep: separators,
			result: match.EveryOf{Matchers: match.Matchers{
				match.Min{3},
				match.Contains{separators, true},
			}},
		},
		{
			ast:    pattern(&nodeAny{}, &nodeSingle{}, &nodeSingle{}, &nodeSingle{}),
			result: match.Min{3},
		},
		{
			ast: pattern(&nodeAny{}, &nodeText{text: "abc"}, &nodeSingle{}),
			sep: separators,
			result: match.NewBTree(
				match.Row{
					Matchers: match.Matchers{
						match.NewText("abc"),
						match.Single{separators},
					},
					RunesLength: 4,
				},
				match.Any{separators},
				nil,
			),
		},
		{
			ast: pattern(&nodeSuper{}, &nodeSingle{}, &nodeText{text: "abc"}, &nodeSingle{}),
			sep: separators,
			result: match.NewBTree(
				match.Row{
					Matchers: match.Matchers{
						match.Single{separators},
						match.NewText("abc"),
						match.Single{separators},
					},
					RunesLength: 5,
				},
				match.Super{},
				nil,
			),
		},
		{
			ast:    pattern(&nodeAny{}, &nodeText{text: "abc"}),
			result: match.Suffix{"abc"},
		},
		{
			ast:    pattern(&nodeText{text: "abc"}, &nodeAny{}),
			result: match.Prefix{"abc"},
		},
		{
			ast:    pattern(&nodeText{text: "abc"}, &nodeAny{}, &nodeText{text: "def"}),
			result: match.PrefixSuffix{"abc", "def"},
		},
		{
			ast:    pattern(&nodeAny{}, &nodeAny{}, &nodeAny{}, &nodeText{text: "abc"}, &nodeAny{}, &nodeAny{}),
			result: match.Contains{"abc", false},
		},
		{
			ast: pattern(&nodeAny{}, &nodeAny{}, &nodeAny{}, &nodeText{text: "abc"}, &nodeAny{}, &nodeAny{}),
			sep: separators,
			result: match.NewBTree(
				match.NewText("abc"),
				match.Any{separators},
				match.Any{separators},
			),
		},
		{
			ast: pattern(&nodeSuper{}, &nodeSingle{}, &nodeText{text: "abc"}, &nodeSuper{}, &nodeSingle{}),
			result: match.NewBTree(
				match.NewText("abc"),
				match.Min{1},
				match.Min{1},
			),
		},
		{
			ast:    pattern(anyOf(&nodeText{text: "abc"})),
			result: match.NewText("abc"),
		},
		{
			ast:    pattern(anyOf(pattern(anyOf(pattern(&nodeText{text: "abc"}))))),
			result: match.NewText("abc"),
		},
		{
			ast: pattern(anyOf(
				pattern(
					&nodeText{text: "abc"},
					&nodeSingle{},
				),
				pattern(
					&nodeText{text: "abc"},
					&nodeList{chars: "def"},
				),
				pattern(
					&nodeText{text: "abc"},
				),
				pattern(
					&nodeText{text: "abc"},
				),
			)),
			result: match.NewBTree(
				match.NewText("abc"),
				nil,
				match.AnyOf{Matchers: match.Matchers{
					match.Single{},
					match.List{List: "def"},
					match.Nothing{},
				}},
			),
		},
		{
			ast: pattern(
				&nodeRange{lo: 'a', hi: 'z'},
				&nodeRange{lo: 'a', hi: 'x', not: true},
				&nodeAny{},
			),
			result: match.NewBTree(
				match.Row{
					Matchers: match.Matchers{
						match.Range{Lo: 'a', Hi: 'z'},
						match.Range{Lo: 'a', Hi: 'x', Not: true},
					},
					RunesLength: 2,
				},
				nil,
				match.Super{},
			),
		},
		{
			ast: pattern(anyOf(
				pattern(
					&nodeText{text: "abc"},
					&nodeList{chars: "abc"},
					&nodeText{text: "ghi"},
				),
				pattern(
					&nodeText{text: "abc"},
					&nodeList{chars: "def"},
					&nodeText{text: "ghi"},
				),
			)),
			result: match.Row{
				RunesLength: 7,
				Matchers: match.Matchers{
					match.NewText("abc"),
					match.AnyOf{Matchers: match.Matchers{
						match.List{List: "abc"},
						match.List{List: "def"},
					}},
					match.NewText("ghi"),
				},
			},
		},
		//				{
		//			ast: pattern(
		//				anyOf(&nodeText{text: "a"}, &nodeText{text: "b"}),
		//				anyOf(&nodeText{text: "c"}, &nodeText{text: "d"}),
		//			),
		//			result: match.AnyOf{Matchers: match.Matchers{
		//				match.Row{Matchers: match.Matchers{match.Raw{"a"}, match.Raw{"c", 1}}},
		//				match.Row{Matchers: match.Matchers{match.Raw{"a"}, match.Raw{"d"}}},
		//				match.Row{Matchers: match.Matchers{match.Raw{"b"}, match.Raw{"c", 1}}},
		//				match.Row{Matchers: match.Matchers{match.Raw{"b"}, match.Raw{"d"}}},
		//			}},
		//		},
	} {
		prog, err := compile(test.ast, test.sep)
		if err != nil {
			t.Errorf("compilation error: %s", err)
			continue
		}

		if !reflect.DeepEqual(prog, test.result) {
			t.Errorf("#%d results are not equal:\nexp: %s,\nact: %s", id, test.result, prog)
			continue
		}
	}
}

const complexityString = "abcd"

//func BenchmarkComplexityAny(b *testing.B) {
//	m := match.Any{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityContains(b *testing.B) {
//	m := match.Contains{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityList(b *testing.B) {
//	m := match.List{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityMax(b *testing.B) {
//	m := match.Max{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityMin(b *testing.B) {
//	m := match.Min{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityNothing(b *testing.B) {
//	m := match.Nothing{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityPrefix(b *testing.B) {
//	m := match.Prefix{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityPrefixSuffix(b *testing.B) {
//	m := match.PrefixSuffix{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityRange(b *testing.B) {
//	m := match.Range{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityRow(b *testing.B) {
//	m := match.Row{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexitySingle(b *testing.B) {
//	m := match.Single{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexitySuffix(b *testing.B) {
//	m := match.Suffix{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexitySuper(b *testing.B) {
//	m := match.Super{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityText(b *testing.B) {
//	m := match.Text{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityAnyOf(b *testing.B) {
//	m := match.AnyOf{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityBTree(b *testing.B) {
//	m := match.NewBTree(match.NewText("abc"), match.NewText("d"), match.NewText("e"))
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
//func BenchmarkComplexityEveryOf(b *testing.B) {
//	m := match.EveryOf{}
//	for i := 0; i < b.N; i++ {
//		_ = m.Match(complexityString)
//		_, _ = m.Index(complexityString)
//	}
//}
