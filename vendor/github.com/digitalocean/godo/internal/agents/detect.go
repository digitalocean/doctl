// Package agents identifies the AI agent, if any, driving the current
// process. The rules live in agents.json, which is shared with the other
// DigitalOcean clients; see agents.schema.json for its format.
package agents

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// supportedVersion is the agents.json format this package can read.
const supportedVersion = 1

//go:embed agents.json
var agentsJSON []byte

type rules struct {
	Version     int    `json:"version"`
	AIAgentVar  string `json:"aiAgentVar"`
	NamePattern string `json:"namePattern"`
	Agents      []rule `json:"agents"`

	namePattern *regexp.Regexp
}

type rule struct {
	Name  string    `json:"name"`
	Match condition `json:"match"`
}

type condition struct {
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	Value      string      `json:"value"`
	Conditions []condition `json:"conditions"`
}

func parse(data []byte) (*rules, error) {
	r := new(rules)
	if err := json.Unmarshal(data, r); err != nil {
		return nil, err
	}
	if r.Version != supportedVersion {
		return nil, fmt.Errorf("agents: unsupported agents.json version %d, want %d", r.Version, supportedVersion)
	}
	re, err := regexp.Compile(r.NamePattern)
	if err != nil {
		return nil, fmt.Errorf("agents: invalid namePattern: %w", err)
	}
	r.namePattern = re
	return r, nil
}

// detect returns the name of the AI agent described by the environment, or ""
// if none matches. A valid aiAgentVar value wins over the rule list.
func (r *rules) detect(lookup func(string) (string, bool)) string {
	if r.AIAgentVar != "" {
		if v, ok := lookup(r.AIAgentVar); ok {
			if v = strings.TrimSpace(v); r.namePattern.MatchString(v) {
				return v
			}
		}
	}
	for _, agent := range r.Agents {
		if agent.Match.matches(lookup) {
			return agent.Name
		}
	}
	return ""
}

func (c condition) matches(lookup func(string) (string, bool)) bool {
	switch c.Type {
	case "anyOf":
		for _, sub := range c.Conditions {
			if sub.matches(lookup) {
				return true
			}
		}
		return false
	case "allOf":
		for _, sub := range c.Conditions {
			if !sub.matches(lookup) {
				return false
			}
		}
		return true
	case "env_set":
		v, ok := lookup(c.Name)
		return ok && v != ""
	case "env_value":
		v, ok := lookup(c.Name)
		return ok && v == c.Value
	default:
		return false
	}
}

var loadRules = sync.OnceValues(func() (*rules, error) {
	return parse(agentsJSON)
})

// Detect returns the name of the AI agent driving the process, or "" if none
// is detected. The rules are parsed once; the environment is read on every
// call. Detection must never break the client, so unreadable rules mean no
// agent.
func Detect() string {
	r, err := loadRules()
	if err != nil {
		return ""
	}
	return r.detect(os.LookupEnv)
}
