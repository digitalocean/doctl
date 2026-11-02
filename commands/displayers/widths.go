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
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// columnCount reports how many columns the table has.
func columnCount(headers []string, rows [][]string) int {
	count := len(headers)
	for _, row := range rows {
		if len(row) > count {
			count = len(row)
		}
	}

	return count
}

// contentBudget is the width left for values once the chrome between columns is
// paid for. It returns 0 when the table is unconstrained, and never less than 1
// otherwise, so a budget too small to honor still asks for narrowing.
func contentBudget(maxWidth, count int, boxed bool) int {
	if maxWidth <= 0 || count == 0 {
		return 0
	}

	chrome := columnGap * (count - 1)
	if boxed {
		// A rule left of every column plus one closing the row, and two pads.
		chrome = count + 1 + 2*cellPad*count
	}

	if budget := maxWidth - chrome; budget > 1 {
		return budget
	}

	return 1
}

// columnWidths measures how wide each column has to be to hold its header and
// its widest value.
//
// Columns are never narrowed. Half a value is a different value - half an ID
// names nothing, half an address routes nowhere - so a table too wide for the
// terminal changes layout rather than losing text. What decides that is
// fitsBudget, on these widths.
func columnWidths(headers []string, rows [][]string) []int {
	count := columnCount(headers, rows)
	if count == 0 {
		return nil
	}

	widths := make([]int, count)
	for i, header := range headers {
		widths[i] = ansi.StringWidth(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := ansi.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	return widths
}

// fitsBudget reports whether columns of these widths fit budget, a budget of 0
// meaning the output is unconstrained and everything fits.
func fitsBudget(widths []int, budget int) bool {
	if budget <= 0 {
		return true
	}

	total := 0
	for _, w := range widths {
		total += w
	}

	return total <= budget
}

// isBreak reports whether a value can be cut at r and still read as a value.
func isBreak(r rune) bool {
	return unicode.IsSpace(r) || r == ','
}
