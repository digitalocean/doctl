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

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// installHelpRenderer replaces Cobra's default help/usage with doctl's shared
// Next Gen layout. It is installed once on the root command and inherited by
// every subcommand.
func installHelpRenderer() {
	DoitCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		env := uiEnv()
		_ = renderStyledHelp(cmd, cmd.OutOrStdout(), env, false)
	})
	DoitCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		env := uiEnv()
		return renderStyledHelp(cmd, cmd.OutOrStderr(), env, true)
	})
}

// helpPainter applies the shared palette to help text. Help normally writes to
// Out; usage-after-error writes to Err, so the same layout can use either
// stream's color gate.
type helpPainter struct {
	env   ui.Env
	onErr bool
}

func (p helpPainter) paint(text string, c lipgloss.TerminalColor, bold bool) string {
	style := lipgloss.NewStyle().Foreground(c)
	if bold {
		style = style.Bold(true)
	}
	if p.onErr {
		return p.env.SprintErr(style, text)
	}
	return p.env.Sprint(style, text)
}

func (p helpPainter) bold(text string) string {
	style := lipgloss.NewStyle().Bold(true)
	if p.onErr {
		if !p.env.ErrStyle {
			return text
		}
		return p.env.SprintErr(style, text)
	}
	if !p.env.Style {
		return text
	}
	return p.env.Sprint(style, text)
}

func (p helpPainter) section(title string) string {
	return p.paint(title, ui.ColorInfo, true)
}

func (p helpPainter) accent(text string) string {
	return p.paint(text, ui.ColorInfo, false)
}

func (p helpPainter) requiredBadge(text string) string {
	return p.paint(text, ui.ColorWarning, true)
}

func (p helpPainter) successBadge(text string) string {
	return p.paint(text, ui.ColorSuccess, false)
}

func (p helpPainter) dim(text string) string {
	return p.paint(text, ui.ColorMuted, false)
}

func (p helpPainter) hint(text string) string {
	lead := p.env.Glyphs().Hint + " "
	if strings.HasPrefix(strings.ToLower(text), "run ") {
		return p.dim(lead+"run ") + p.bold(strings.TrimSpace(text[4:]))
	}
	return p.dim(lead + text)
}

func renderStyledHelp(cmd *cobra.Command, w io.Writer, env ui.Env, onErr bool) error {
	if cmd == nil {
		return nil
	}
	// Help runs before PersistentPreRun on a bare --help, so resolve Env here
	// when installOutputPolicy has not yet run.
	if resolvedEnv == nil {
		env = detectUIEnv()
	}

	p := helpPainter{env: env, onErr: onErr}
	var b strings.Builder

	writeHelpHeader(&b, p, cmd)
	writeHelpUsage(&b, p, cmd)
	writeHelpAliases(&b, p, cmd)
	writeHelpCommands(&b, p, cmd)
	writeHelpFlags(&b, p, cmd)
	writeHelpExample(&b, p, cmd)
	writeHelpFooter(&b, p, cmd)

	_, err := fmt.Fprint(w, b.String())
	return err
}

func writeHelpHeader(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	short := strings.TrimSpace(cmd.Short)
	long := strings.TrimSpace(cmd.Long)

	switch {
	case short != "" && long != "" && long != short:
		fmt.Fprintf(b, "%s\n\n", p.bold(short))
		fmt.Fprintf(b, "%s\n\n", long)
	case long != "":
		fmt.Fprintf(b, "%s\n\n", long)
	case short != "":
		fmt.Fprintf(b, "%s\n\n", p.bold(short))
	}
}

func writeHelpUsage(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	if !cmd.Runnable() && !cmd.HasAvailableSubCommands() {
		return
	}
	fmt.Fprintf(b, "%s\n", p.section("USAGE"))
	// Match Cobra's default template: runnable commands keep UseLine (with
	// [flags]); parents that only dispatch subcommands advertise [command].
	if cmd.Runnable() {
		fmt.Fprintf(b, "  %s\n", p.accent(strings.TrimSpace(cmd.UseLine())))
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(b, "  %s\n", p.accent(cmd.CommandPath()+" [command]"))
	}
	b.WriteByte('\n')
}

func writeHelpAliases(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	if len(cmd.Aliases) == 0 {
		return
	}
	fmt.Fprintf(b, "%s\n", p.section("ALIASES"))
	fmt.Fprintf(b, "  %s\n\n", p.dim(cmd.NameAndAliases()))
}

func writeHelpCommands(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	if !cmd.HasAvailableSubCommands() {
		return
	}

	all := availableCommands(cmd)
	pad := 0
	for _, c := range all {
		if n := len(c.Name()); n > pad {
			pad = n
		}
	}
	if pad < 12 {
		pad = 12
	}

	groups := cmd.Groups()
	if len(groups) == 0 {
		fmt.Fprintf(b, "%s\n", p.section("COMMANDS"))
		writeCommandRows(b, p, all, pad)
		b.WriteByte('\n')
		return
	}

	for _, g := range groups {
		rows := availableCommandsInGroup(cmd, g.ID)
		if len(rows) == 0 {
			continue
		}
		title := strings.TrimSuffix(strings.TrimSpace(g.Title), ":")
		fmt.Fprintf(b, "%s\n", p.section(strings.ToUpper(title)))
		writeCommandRows(b, p, rows, pad)
		b.WriteByte('\n')
	}

	if extra := availableCommandsInGroup(cmd, ""); len(extra) > 0 {
		fmt.Fprintf(b, "%s\n", p.section("ADDITIONAL COMMANDS"))
		writeCommandRows(b, p, extra, pad)
		b.WriteByte('\n')
	}
}

func availableCommands(cmd *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() || c.Name() == "help" {
			out = append(out, c)
		}
	}
	return out
}

func availableCommandsInGroup(cmd *cobra.Command, groupID string) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range availableCommands(cmd) {
		if c.GroupID == groupID {
			out = append(out, c)
		}
	}
	return out
}

func writeCommandRows(b *strings.Builder, p helpPainter, cmds []*cobra.Command, pad int) {
	for _, c := range cmds {
		name := fmt.Sprintf("%-*s", pad, c.Name())
		fmt.Fprintf(b, "  %s %s\n", p.bold(name), strings.TrimSpace(c.Short))
	}
}

type helpFlag struct {
	flag     *pflag.Flag
	purpose  string
	hint     string
	required bool
	output   bool
}

func writeHelpFlags(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	local := collectHelpFlags(cmd.LocalFlags())
	inherited := collectHelpFlags(cmd.InheritedFlags())

	var required, options, output []helpFlag
	for _, f := range local {
		switch {
		case f.required:
			required = append(required, f)
		case f.output:
			output = append(output, f)
		default:
			options = append(options, f)
		}
	}

	writeFlagSection(b, p, "REQUIRED", required, true)

	optionsTitle := "OPTIONS"
	if cmd.Parent() == nil {
		// Root flags are the process-wide defaults users think of as global.
		optionsTitle = "FLAGS"
	}
	writeFlagSection(b, p, optionsTitle, options, false)
	writeFlagSection(b, p, "OUTPUT", output, false)
	writeFlagSection(b, p, "GLOBAL FLAGS", inherited, false)
}

func collectHelpFlags(fs *pflag.FlagSet) []helpFlag {
	if fs == nil {
		return nil
	}
	var out []helpFlag
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		// Skip the help flag itself; every command already exposes -h/--help.
		if f.Name == "help" {
			return
		}
		hf := helpFlag{
			flag:     f,
			required: isRequiredFlag(f) || requiredUsageSuffix.MatchString(f.Usage),
			output:   f.Name == doctl.ArgFormat || f.Name == doctl.ArgNoHeader,
		}
		purpose := flagAnnotation(f, annoFlagPurpose)
		hint := flagAnnotation(f, annoFlagHint)
		usage := cleanFlagUsage(f.Usage)
		if purpose == "" && usage != "" {
			purpose = shortenUsage(usage)
		} else if purpose == "" {
			purpose = usage
		}
		if hint == "" {
			hint = extractHintFromUsage(usage)
		}
		// When purpose is a shortened usage and the full usage was only that
		// sentence, avoid repeating a near-identical line under the flag.
		hf.purpose = purpose
		hf.hint = hint
		out = append(out, hf)
	})
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].flag.Name < out[j].flag.Name
	})
	return out
}

func writeFlagSection(b *strings.Builder, p helpPainter, title string, flags []helpFlag, markRequired bool) {
	if len(flags) == 0 {
		return
	}
	fmt.Fprintf(b, "%s\n", p.section(title))
	for _, hf := range flags {
		writeHelpFlag(b, p, hf, markRequired)
	}
	b.WriteByte('\n')
}

func writeHelpFlag(b *strings.Builder, p helpPainter, hf helpFlag, markRequired bool) {
	f := hf.flag
	label := formatFlagLabel(f)
	line := "  " + p.bold(label)

	if markRequired || hf.required {
		badge := " required"
		if p.env.ASCII {
			badge = " (required)"
		} else {
			badge = " " + ui.GlyphAsterisk + " required"
		}
		line += p.requiredBadge(badge)
	}

	if def := defaultFlagValue(f); def != "" {
		line += p.dim("  [default: " + def + "]")
	}

	fmt.Fprintf(b, "%s\n", line)

	if hf.purpose != "" {
		fmt.Fprintf(b, "      %s\n", hf.purpose)
	}
	if hf.hint != "" {
		fmt.Fprintf(b, "      %s\n", p.hint(hf.hint))
	}
}

func formatFlagLabel(f *pflag.Flag) string {
	var b strings.Builder
	if f.Shorthand != "" && f.ShorthandDeprecated == "" {
		b.WriteString("-")
		b.WriteString(f.Shorthand)
		b.WriteString(", --")
	} else {
		b.WriteString("--")
	}
	b.WriteString(f.Name)

	typename := flagTypeName(f)
	if typename != "" && typename != "bool" {
		b.WriteByte(' ')
		b.WriteString(typename)
	}
	return b.String()
}

func flagTypeName(f *pflag.Flag) string {
	// Prefer the backticked placeholder in Usage, matching Cobra/pflag.
	if start := strings.IndexByte(f.Usage, '`'); start >= 0 {
		if end := strings.IndexByte(f.Usage[start+1:], '`'); end >= 0 {
			return f.Usage[start+1 : start+1+end]
		}
	}
	switch t := f.Value.Type(); t {
	case "bool":
		return ""
	case "stringSlice", "stringArray":
		return "strings"
	default:
		return t
	}
}

func defaultFlagValue(f *pflag.Flag) string {
	switch f.DefValue {
	case "", "false", "[]", "0", "0s", "0.0":
		return ""
	default:
		return f.DefValue
	}
}

func writeHelpExample(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	ex := strings.TrimSpace(cmd.Example)
	if ex == "" {
		return
	}
	fmt.Fprintf(b, "%s\n", p.section("EXAMPLES"))
	for _, line := range strings.Split(ex, "\n") {
		fmt.Fprintf(b, "%s\n", formatExampleLine(p, line))
	}
	b.WriteByte('\n')
}

// formatExampleLine highlights every real doctl invocation on the line, whether
// bare or embedded mid-sentence, without greening trailing prose or English
// phrases like "initializes doctl with…".
func formatExampleLine(p helpPainter, line string) string {
	trimmedRight := strings.TrimRightFunc(line, unicode.IsSpace)
	content := strings.TrimLeft(trimmedRight, " \t")
	indent := trimmedRight[:len(trimmedRight)-len(content)]

	var b strings.Builder
	b.WriteString(indent)
	remaining := content
	for {
		idx := indexDoctlInvocation(remaining)
		if idx < 0 {
			b.WriteString(remaining)
			return b.String()
		}
		b.WriteString(remaining[:idx])
		invocation, suffix := splitDoctlInvocation(remaining[idx:])
		b.WriteString(p.successBadge(invocation))
		remaining = suffix
	}
}

// indexDoctlInvocation returns the start of the next real doctl CLI invocation
// in s, or -1. Prose that merely mentions "doctl" (e.g. "initializes doctl with")
// is skipped by requiring the following token to be a known root command.
func indexDoctlInvocation(s string) int {
	roots := doctlRootCommandNames()
	for i := 0; i+6 <= len(s); i++ {
		if !strings.HasPrefix(s[i:], "doctl ") {
			continue
		}
		if i > 0 {
			prev := s[i-1]
			if unicode.IsLetter(rune(prev)) || unicode.IsDigit(rune(prev)) || prev == '_' || prev == '-' {
				continue
			}
		}
		arg := firstCLIToken(s[i+len("doctl "):])
		if arg == "" || !roots[arg] {
			continue
		}
		return i
	}
	return -1
}

// doctlRootCommandNames is the set of top-level doctl subcommands, used to tell
// real invocations apart from English that happens to contain "doctl ".
func doctlRootCommandNames() map[string]bool {
	names := make(map[string]bool)
	for _, c := range DoitCmd.Commands() {
		if c.IsAvailableCommand() || c.Name() == "help" {
			names[c.Name()] = true
			for _, a := range c.Aliases {
				names[a] = true
			}
		}
	}
	// Always allow these even if hidden/grouped unusually during tests.
	names["help"] = true
	names["completion"] = true
	names["version"] = true
	return names
}

func firstCLIToken(s string) string {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return ""
	}
	end := 0
	for end < len(s) {
		r, size := utf8.DecodeRuneInString(s[end:])
		if unicode.IsSpace(r) || r == '"' || r == '\'' {
			break
		}
		end += size
	}
	return s[:end]
}

// splitDoctlInvocation separates a doctl command from trailing prose on the same
// line. Many Example strings continue after the command with a new sentence
// ("…: doctl foo bar. Note that…"); greening through EOL would paint that note
// and, because EXAMPLES is last in help, look like color bled to the end.
func splitDoctlInvocation(s string) (invocation, suffix string) {
	end := len(s)
	for i := 0; i < len(s)-1; i++ {
		if s[i] != '.' && s[i] != '!' && s[i] != '?' {
			continue
		}
		if s[i+1] != ' ' {
			continue
		}
		j := i + 2
		for j < len(s) && s[j] == ' ' {
			j++
		}
		if j < len(s) {
			r, _ := utf8.DecodeRuneInString(s[j:])
			if unicode.IsUpper(r) {
				end = i
				break
			}
		}
	}

	invocation = strings.TrimSpace(s[:end])
	suffix = s[end:]
	return invocation, suffix
}

func writeHelpFooter(b *strings.Builder, p helpPainter, cmd *cobra.Command) {
	if !cmd.HasAvailableSubCommands() {
		return
	}
	fmt.Fprintf(b, "%s\n", p.dim(fmt.Sprintf(
		"Use \"%s [command] --help\" for more information about a command.",
		cmd.CommandPath(),
	)))
}
