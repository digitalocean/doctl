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

package displayers

import (
	"bytes"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/digitalocean/doctl/internal/ui"
)

// renderer draws one table. It holds what every layout needs - where to write,
// what the terminal can do, and the rules and glyphs that follow - so a layout
// takes only the data it lays out, and a new layout is a method here.
type renderer struct {
	buf      *bytes.Buffer
	env      ui.Env
	border   lipgloss.Border
	ellipsis string
}

func newRenderer(buf *bytes.Buffer, env ui.Env) renderer {
	return renderer{
		buf:      buf,
		env:      env,
		border:   borderFor(env),
		ellipsis: env.Glyphs().Ellipsis,
	}
}

// borderFor returns the rules both framed layouts draw with, so a table and a
// card cannot drift apart.
func borderFor(env ui.Env) lipgloss.Border {
	if env.ASCII {
		return lipgloss.ASCIIBorder()
	}

	return lipgloss.RoundedBorder()
}

// frameStyle paints the rules a table or card is drawn in. The frame takes the
// success color whatever state the resource is in; the status field keeps its own.
func frameStyle(env ui.Env) lipgloss.Style {
	return env.NewStyle().Foreground(ui.ColorSuccess)
}

// vertical is the painted rule closing either side of a framed row.
func (r renderer) vertical() string {
	return r.env.Sprint(frameStyle(r.env), r.border.Left)
}

// fit narrows cell to width, ending it in the ellipsis when it has to cut.
func (r renderer) fit(cell string, width int) string {
	if ansi.StringWidth(cell) > width {
		return ansi.Truncate(cell, width, r.ellipsis)
	}

	return cell
}

// row writes one row of the plain layout, padding cells to their column. Widths
// are measured in terminal cells, so styled and wide values stay aligned.
func (r renderer) row(cells []string, widths []int, paint painter) {
	for i, cell := range cells {
		cell = r.fit(cell, widths[i])
		padding := widths[i] - ansi.StringWidth(cell) + columnGap

		r.buf.WriteString(paintCell(paint, i, cell))

		if i < len(cells)-1 && padding > 0 {
			r.buf.WriteString(strings.Repeat(" ", padding))
		}
	}
	r.buf.WriteString("\n")
}
