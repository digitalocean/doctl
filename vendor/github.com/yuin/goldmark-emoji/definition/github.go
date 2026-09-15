package definition

import "sync"

//go:generate go run ../_tools github -o ../_tools/github.json
//go:generate go run ../_tools emb-structs -o ./github.gen.go -i ../_tools/github.json

var github Emojis
var githubOnce sync.Once

func Github(opts ...EmojisOption) Emojis {
	githubOnce.Do(func() {
		lst := make([]Emoji, _githubLength)
		emojiMap := make(map[string]*Emoji, _githubLength)

		cName := 0
		cUnicode := 0
		cShortNames := 0
		for i := 0; i < _githubLength; i++ {
			tName := cName + int(_githubNameIndex[i])
			tUnicode := cUnicode + int(_githubUnicodeIndex[i])
			tShortNames := cShortNames + int(_githubShortNamesIndex[i])

			name := _githubName[cName:tName]
			e := &lst[i]
			e.Name = name
			e.Unicode = _githubUnicode[cUnicode:tUnicode]
			e.ShortNames = _githubShortNames[cShortNames:tShortNames]
			for _, s := range e.ShortNames {
				emojiMap[s] = e
			}

			cName = tName
			cUnicode = tUnicode
			cShortNames = tShortNames
		}
		github = &emojis{
			list: lst,
			m:    emojiMap,
		}
	})

	if len(opts) == 0 {
		return github
	}

	m := github.Clone()
	for _, opt := range opts {
		opt(m)
	}

	return m
}
