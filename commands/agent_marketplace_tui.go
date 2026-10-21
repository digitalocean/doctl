/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

// The skill picker `marketplace init` shows: a checklist of the skills found
// in the local Claude Code setup with a live size budget, so the author sees
// exactly what will be sent before anything leaves the machine.

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var errMKPickerCancelled = errors.New("cancelled; nothing was created or changed")

type mkPickerItem struct {
	skill    mkSkill
	selected bool
}

type mkPicker struct {
	items     []mkPickerItem
	cursor    int
	done      bool
	cancelled bool
	note      string
}

func newMKPicker(skills []mkSkill, preselect map[string]bool) *mkPicker {
	p := &mkPicker{}
	for _, s := range skills {
		p.items = append(p.items, mkPickerItem{skill: s, selected: s.Problem == "" && preselect[s.Name]})
	}
	return p
}

func (p *mkPicker) totals() (n, bytes int) {
	for _, it := range p.items {
		if it.selected {
			n++
			bytes += it.skill.Bytes()
		}
	}
	return n, bytes
}

func (p *mkPicker) Init() tea.Cmd { return nil }

func (p *mkPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}
	p.note = ""
	switch k.String() {
	case "ctrl+c", "esc", "q":
		p.cancelled = true
		return p, tea.Quit
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "j":
		if p.cursor < len(p.items)-1 {
			p.cursor++
		}
	case " ", "x":
		p.toggle(p.cursor)
	case "n":
		for i := range p.items {
			p.items[i].selected = false
		}
	case "enter":
		if n, _ := p.totals(); n == 0 {
			p.note = "select at least one skill"
			return p, nil
		}
		p.done = true
		return p, tea.Quit
	}
	return p, nil
}

func (p *mkPicker) toggle(i int) {
	if len(p.items) == 0 {
		return
	}
	it := &p.items[i]
	if it.skill.Problem != "" {
		p.note = it.skill.Name + " cannot be published: " + it.skill.Problem
		return
	}
	if !it.selected {
		n, b := p.totals()
		if n+1 > mkMaxSkills {
			p.note = fmt.Sprintf("a listing holds at most %d skills", mkMaxSkills)
			return
		}
		if b+it.skill.Bytes() > mkMaxSkillTotalBytes {
			p.note = fmt.Sprintf("adding %s would pass the %s total", it.skill.Name, humanBytes(uint64(mkMaxSkillTotalBytes)))
			return
		}
	}
	it.selected = !it.selected
}

func (p *mkPicker) View() string {
	if p.done || p.cancelled {
		return ""
	}
	var b strings.Builder
	b.WriteString(boldColor("Choose the skills to publish", colHighlight) + "\n")
	b.WriteString(colorize("Only the selected SKILL.md files are sent. Support files, hooks and local MCP servers stay on your machine.", colMuted) + "\n\n")
	for i, it := range p.items {
		cursor := "  "
		if i == p.cursor {
			cursor = colorize("> ", colHighlight)
		}
		box := "[ ]"
		if it.selected {
			box = colorize("[x]", colSuccess)
		}
		name := fmt.Sprintf("%-28s", mkTruncate(it.skill.Name, 28))
		meta := fmt.Sprintf("%8s  %-7s", humanBytes(uint64(it.skill.Bytes())), it.skill.Source)
		line := cursor + box + " " + name + " " + colorize(meta, colMuted)
		if it.skill.Problem != "" {
			line = cursor + colorize("[-] "+name+" "+meta+"  "+it.skill.Problem, colMuted)
		} else if it.skill.ExtraFiles > 0 {
			line += colorize(fmt.Sprintf("  +%d support files not carried", it.skill.ExtraFiles), colWarning)
		}
		b.WriteString(line + "\n")
	}
	if len(p.items) > 0 {
		desc := p.items[p.cursor].skill.Description
		b.WriteString("\n" + colorize(mkTruncate(desc, 160), colMuted) + "\n")
	}
	n, total := p.totals()
	b.WriteString("\n" + mkBudgetBar(total, mkMaxSkillTotalBytes, 30) +
		fmt.Sprintf("  %d/%d skills · %s of %s\n", n, mkMaxSkills, humanBytes(uint64(total)), humanBytes(uint64(mkMaxSkillTotalBytes))))
	if p.note != "" {
		b.WriteString(colorize(p.note, colWarning) + "\n")
	}
	b.WriteString(colorize("↑/↓ move · space select · n clear · enter continue · q cancel", colMuted) + "\n")
	return b.String()
}

func mkBudgetBar(used, limit, width int) string {
	filled := 0
	if limit > 0 {
		filled = used * width / limit
	}
	if filled > width {
		filled = width
	}
	return colorize(strings.Repeat("█", filled), colSuccess) + colorize(strings.Repeat("░", width-filled), colMuted)
}

// runMKPicker shows the checklist and returns the chosen skill names.
func runMKPicker(skills []mkSkill, preselect map[string]bool) ([]string, error) {
	m, err := tea.NewProgram(newMKPicker(skills, preselect)).StartReturningModel()
	if err != nil {
		return nil, err
	}
	p := m.(*mkPicker)
	if p.cancelled || !p.done {
		return nil, errMKPickerCancelled
	}
	var out []string
	for _, it := range p.items {
		if it.selected {
			out = append(out, it.skill.Name)
		}
	}
	return out, nil
}
