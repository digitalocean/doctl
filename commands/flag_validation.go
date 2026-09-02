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
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/digitalocean/doctl/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// Annotation keys stored on pflag for enriched pre-run validation.
	annoFlagPurpose  = "doctl_flag_purpose"
	annoFlagHint     = "doctl_flag_hint"
	annoFlagRequired = "doctl_flag_required"
	annoFlagViperKey = "doctl_flag_viper_key"

	// cobraRequiredAnno is the annotation MarkFlagRequired sets. Still
	// honoured for flags marked that way outside requiredOpt().
	cobraRequiredAnno = cobra.BashCompOneRequiredFlag
)

// FlagIssue describes one flag validation problem.
type FlagIssue struct {
	Flag    string `json:"flag"`
	Problem string `json:"problem"`
	Purpose string `json:"purpose,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

// FlagValidationError aggregates every flag problem found in one pre-run pass.
type FlagValidationError struct {
	Command string      `json:"command"`
	Issues  []FlagIssue `json:"issues"`
}

func (e *FlagValidationError) Error() string {
	return e.format(ui.NewStyle(ui.Plain(os.Stdout, os.Stderr)))
}

// Display renders the validation error using the Next-Gen terminal design
// system (colored error label, bold flags/commands, dim hints).
func (e *FlagValidationError) Display() string {
	env := ui.Detect(os.Stdout, os.Stderr)
	return e.format(ui.NewStyle(env))
}

func (e *FlagValidationError) format(style ui.Style) string {
	if e == nil || len(e.Issues) == 0 {
		return "flag validation failed"
	}

	missing := make([]FlagIssue, 0, len(e.Issues))
	other := make([]FlagIssue, 0, len(e.Issues))
	for _, issue := range e.Issues {
		if isMissingRequiredProblem(issue.Problem) {
			missing = append(missing, issue)
		} else {
			other = append(other, issue)
		}
	}

	var b strings.Builder
	if len(missing) > 0 {
		if len(missing) == 1 {
			fmt.Fprintf(&b, "%s Missing required flag for %s.\n\n", style.ErrorLabel(), e.Command)
		} else {
			fmt.Fprintf(&b, "%s Missing %d required flags for %s.\n\n", style.ErrorLabel(), len(missing), e.Command)
		}
		for _, issue := range missing {
			writeFlagIssue(&b, style, issue)
		}
	}

	if len(other) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		if len(other) == 1 {
			fmt.Fprintf(&b, "%s Invalid flag for %s.\n\n", style.ErrorLabel(), e.Command)
		} else {
			fmt.Fprintf(&b, "%s Invalid flags for %s (%d problems).\n\n", style.ErrorLabel(), e.Command, len(other))
		}
		for _, issue := range other {
			writeFlagIssue(&b, style, issue)
		}
	}

	fmt.Fprintf(&b, "  Run  %s  for usage.", style.Bold(e.Command+" --help"))
	return b.String()
}

func writeFlagIssue(b *strings.Builder, style ui.Style, issue FlagIssue) {
	fmt.Fprintf(b, "  %s\n", style.Bold("--"+issue.Flag))
	if issue.Purpose != "" {
		fmt.Fprintf(b, "      %s\n", issue.Purpose)
	}
	switch {
	case issue.Hint != "":
		fmt.Fprintf(b, "      %s\n", style.PaintCommand("→ "+issue.Hint))
	case strings.Contains(issue.Problem, "was empty"):
		fmt.Fprintf(b, "      %s\n", style.Dim("→ provided but empty"))
	case issue.Problem != "" && !isMissingRequiredProblem(issue.Problem):
		fmt.Fprintf(b, "      %s\n", style.Dim("→ "+issue.Problem))
	}
	b.WriteByte('\n')
}

func isMissingRequiredProblem(problem string) bool {
	return strings.Contains(problem, "required but was not set") ||
		strings.Contains(problem, "required but was empty")
}

var (
	requiredUsageSuffix = regexp.MustCompile(`(?i)\s*\(required\)\s*$`)
	ansiEscapeRE        = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	// Captures discovery commands embedded in flag help text.
	doctlHintRE = regexp.MustCompile("(?i)(?:use the |run[`'\" ]+|via )[`'\"]?(doctl [^`'\".]+)[`'\"]?")
)

func enrichFlagIssue(f *pflag.Flag, issue FlagIssue) FlagIssue {
	if f == nil {
		return issue
	}
	if issue.Purpose == "" {
		issue.Purpose = flagAnnotation(f, annoFlagPurpose)
	}
	if issue.Hint == "" {
		issue.Hint = flagAnnotation(f, annoFlagHint)
	}

	usage := cleanFlagUsage(f.Usage)
	if issue.Purpose == "" && usage != "" {
		issue.Purpose = shortenUsage(usage)
	}
	if issue.Hint == "" {
		if hint := extractHintFromUsage(usage); hint != "" {
			issue.Hint = hint
		}
	}
	return issue
}

func cleanFlagUsage(usage string) string {
	// requiredOpt() appends a colored "(required)" — strip ANSI first, then the suffix.
	usage = ansiEscapeRE.ReplaceAllString(usage, "")
	usage = requiredUsageSuffix.ReplaceAllString(usage, "")
	usage = strings.ReplaceAll(usage, "`", "")
	return strings.TrimSpace(usage)
}

func shortenUsage(usage string) string {
	// Keep the first sentence so the error stays scannable.
	for _, sep := range []string{". ", "? ", "! "} {
		if i := strings.Index(usage, sep); i > 0 {
			return strings.TrimSpace(usage[:i+1])
		}
	}
	if strings.HasSuffix(usage, ".") || strings.HasSuffix(usage, "?") || strings.HasSuffix(usage, "!") {
		return usage
	}
	if len(usage) > 120 {
		return strings.TrimSpace(usage[:117]) + "..."
	}
	return usage
}

func extractHintFromUsage(usage string) string {
	m := doctlHintRE.FindStringSubmatch(usage)
	if len(m) < 2 {
		return ""
	}
	cmd := strings.TrimSpace(m[1])
	cmd = strings.Trim(cmd, "`.\"'")
	// Keep only the doctl command path (drop trailing prose like "command for a list...").
	parts := strings.Fields(cmd)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		lower := strings.ToLower(strings.Trim(p, ".,;:"))
		if lower == "command" || lower == "commands" || lower == "for" || lower == "to" || lower == "and" || lower == "with" {
			break
		}
		out = append(out, strings.Trim(p, ".,;:"))
	}
	if len(out) == 0 {
		return ""
	}
	return "run " + strings.Join(out, " ")
}

// validateCommandFlags runs a single pre-run pass over required flags and
// annotated flag groups, collecting every problem instead of failing one-by-one.
func validateCommandFlags(cmd *cobra.Command) error {
	if cmd == nil || cmd.DisableFlagParsing {
		return nil
	}

	issues := make([]FlagIssue, 0)
	issues = append(issues, collectMissingRequiredFlags(cmd)...)
	issues = append(issues, collectFlagGroupIssues(cmd)...)

	if len(issues) == 0 {
		return nil
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Flag == issues[j].Flag {
			return issues[i].Problem < issues[j].Problem
		}
		return issues[i].Flag < issues[j].Flag
	})

	// Keep help output out of the error path; checkErr prints once.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	return &FlagValidationError{
		Command: cmd.CommandPath(),
		Issues:  issues,
	}
}

func isRequiredFlag(f *pflag.Flag) bool {
	if flagAnnotation(f, annoFlagRequired) == "true" {
		return true
	}
	required, found := f.Annotations[cobraRequiredAnno]
	return found && len(required) > 0 && required[0] == "true"
}

// flagEffectiveValue returns the value doctl would actually use: the pflag
// value (CLI or default), falling back to viper so config.yaml / env values
// satisfy required flags even when pflag.Changed is false.
func flagEffectiveValue(f *pflag.Flag) string {
	if f == nil {
		return ""
	}

	raw := strings.TrimSpace(f.Value.String())
	if !isEmptyFlagRaw(raw) {
		return raw
	}

	if key := flagAnnotation(f, annoFlagViperKey); key != "" {
		if v := strings.TrimSpace(viper.GetString(key)); !isEmptyFlagRaw(v) {
			return v
		}
	}

	return raw
}

func isEmptyFlagRaw(raw string) bool {
	return raw == "" || raw == "[]"
}

func collectMissingRequiredFlags(cmd *cobra.Command) []FlagIssue {
	var issues []FlagIssue
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden || !isRequiredFlag(f) {
			return
		}

		val := flagEffectiveValue(f)
		if !isEmptyFlagRaw(val) {
			return
		}

		problem := "is required but was not set"
		if f.Changed {
			problem = "is required but was empty"
		}

		issues = append(issues, enrichFlagIssue(f, FlagIssue{
			Flag:    f.Name,
			Problem: problem,
			Purpose: flagAnnotation(f, annoFlagPurpose),
			Hint:    flagAnnotation(f, annoFlagHint),
		}))
	})
	return issues
}

func collectFlagGroupIssues(cmd *cobra.Command) []FlagIssue {
	const (
		requiredAsGroup   = "cobra_annotation_required_if_others_set"
		mutuallyExclusive = "cobra_annotation_mutually_exclusive"
	)

	requiredGroups := map[string]map[string]bool{}
	exclusiveGroups := map[string]map[string]bool{}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		recordGroup(cmd.Flags(), f, requiredAsGroup, requiredGroups)
		recordGroup(cmd.Flags(), f, mutuallyExclusive, exclusiveGroups)
	})

	var issues []FlagIssue

	for group, statuses := range requiredGroups {
		unset := unsetFlags(statuses)
		if len(unset) == 0 || len(unset) == len(statuses) {
			continue
		}
		sort.Strings(unset)
		for _, name := range unset {
			f := cmd.Flags().Lookup(name)
			issues = append(issues, enrichFlagIssue(f, FlagIssue{
				Flag:    name,
				Problem: fmt.Sprintf("must be set together with [%s]", strings.ReplaceAll(group, " ", ", ")),
				Purpose: flagAnnotation(f, annoFlagPurpose),
				Hint:    flagAnnotation(f, annoFlagHint),
			}))
		}
	}

	for _, statuses := range exclusiveGroups {
		set := setFlags(statuses)
		if len(set) < 2 {
			continue
		}
		sort.Strings(set)
		for _, name := range set {
			f := cmd.Flags().Lookup(name)
			others := make([]string, 0, len(set)-1)
			for _, n := range set {
				if n != name {
					others = append(others, "--"+n)
				}
			}
			issues = append(issues, enrichFlagIssue(f, FlagIssue{
				Flag:    name,
				Problem: fmt.Sprintf("cannot be combined with %s", strings.Join(others, ", ")),
				Purpose: flagAnnotation(f, annoFlagPurpose),
				Hint:    flagAnnotation(f, annoFlagHint),
			}))
		}
	}

	return issues
}

func recordGroup(fs *pflag.FlagSet, f *pflag.Flag, annotation string, out map[string]map[string]bool) {
	groups, found := f.Annotations[annotation]
	if !found {
		return
	}
	for _, group := range groups {
		names := strings.Split(group, " ")
		if !allFlagsExist(fs, names...) {
			continue
		}
		if out[group] == nil {
			out[group] = map[string]bool{}
			for _, name := range names {
				out[group][name] = false
			}
		}
		out[group][f.Name] = f.Changed
	}
}

func allFlagsExist(fs *pflag.FlagSet, names ...string) bool {
	for _, name := range names {
		if fs.Lookup(name) == nil {
			return false
		}
	}
	return true
}

func unsetFlags(statuses map[string]bool) []string {
	var unset []string
	for name, isSet := range statuses {
		if !isSet {
			unset = append(unset, name)
		}
	}
	return unset
}

func setFlags(statuses map[string]bool) []string {
	var set []string
	for name, isSet := range statuses {
		if isSet {
			set = append(set, name)
		}
	}
	return set
}

func flagAnnotation(f *pflag.Flag, key string) string {
	if f == nil {
		return ""
	}
	vals, ok := f.Annotations[key]
	if !ok || len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}

// flagPurpose attaches a one-line purpose shown in aggregated validation errors.
func flagPurpose(purpose string) flagOpt {
	return func(c *Command, name, key string) {
		_ = c.Flags().SetAnnotation(name, annoFlagPurpose, []string{purpose})
	}
}

// flagHint attaches a discovery hint (e.g. a list command) for validation errors.
func flagHint(hint string) flagOpt {
	return func(c *Command, name, key string) {
		_ = c.Flags().SetAnnotation(name, annoFlagHint, []string{hint})
	}
}
