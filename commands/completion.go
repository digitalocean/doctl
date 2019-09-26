/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	completionLong = `
completion is used to output completion code for bash and zsh shells.

Before using completion features, you have to source completion code
from your .profile or .bashrc/.zshrc file. This is done by adding
following line to one of above files:
	source <(doctl completion SHELL)

Bash users can as well save it to the file and copy it to:
	/etc/bash_completion.d/

Correct arguments for SHELL are: "bash" and "zsh".

Notes:
1) zsh completions requires zsh 5.2 or newer.

2) macOS users have to install bash-completion framework to utilize
completion features. This can be done using homebrew:
	brew install bash-completion

Once installed, you must load bash_completion by adding following
line to your .profile or .bashrc/.zshrc:
	source $(brew --prefix)/etc/bash_completion
`
	doctlLicense = `# Copyright 2018 The Doctl Authors All rights reserved.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#   http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

`
)

// Completion creates the completion command
func Completion() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "completion",
			Short: "completion commands",
			Long:  completionLong,
		},
	}

	cmdBuilderWithInit(cmd, RunCompletionBash, "bash", "generate bash completion code", "generate bash completion code", Writer, false)
	cmdBuilderWithInit(cmd, RunCompletionZsh, "zsh", "generate zsh completion code", "generate zsh completion code", Writer, false)

	return cmd
}

// RunCompletionBash outputs completion code for bash.
func RunCompletionBash(c *CmdConfig) error {
	var buf bytes.Buffer

	_, err := buf.Write([]byte(doctlLicense))
	if err != nil {
		return fmt.Errorf("error while generating bash completion: %v", err)
	}

	err = DoitCmd.GenBashCompletion(&buf)
	if err != nil {
		return fmt.Errorf("error while generating bash completion: %v", err)
	}

	// remove the command "completion" from auto-completion
	code := buf.String()
	code = strings.Replace(code, `commands+=("completion")`, "", -1)

	fmt.Print(code)
	return nil
}

// RunCompletionZsh outputs completion code for zsh shell.
func RunCompletionZsh(c *CmdConfig) error {
	var buf bytes.Buffer

	// zshHead is the header required to declare zsh completion
	zshHead := "#compdef doctl\n"

	// zshInit represents intialization code needed to convert bash completion
	// code to zsh completion.
	zshInit := `
__doctl_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand

	source "$@"
}

__doctl_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift

		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__doctl_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}

__doctl_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?

	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}

__doctl_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}

__doctl_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}

__doctl_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}

__doctl_filedir() {
	local RET OLD_IFS w qw

	__debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi

	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"

	IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"

	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__doctl_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}

__doctl_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
    	printf %q "$1"
    fi
}

autoload -U +X bashcompinit && bashcompinit

# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi

__doctl_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__doctl_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__doctl_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__doctl_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__doctl_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__doctl_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e 's/aliashash\["\(.\{1,\}\)"\]/aliashash[\1]/g' \
	-e "s/\\\$(type${RWORD}/\$(__doctl_type/g" \
	<<'BASH_COMPLETION_EOF'
`

	// zshFinalize is code going to the end of completion file
	// that calls conversion bash to zsh.
	zshFinalize := `
BASH_COMPLETION_EOF
}

__doctl_bash_source <(__doctl_convert_bash_to_zsh)
_complete doctl 2>/dev/null
`

	_, err := buf.Write([]byte(zshHead))
	if err != nil {
		return fmt.Errorf("error while generating zsh completion: %v", err)
	}

	_, err = buf.Write([]byte(doctlLicense))
	if err != nil {
		return fmt.Errorf("error while generating zsh completion: %v", err)
	}

	_, err = buf.Write([]byte(zshInit))
	if err != nil {
		return fmt.Errorf("error while generating zsh completion: %v", err)
	}

	err = DoitCmd.GenBashCompletion(&buf)
	if err != nil {
		return fmt.Errorf("error wheil generating zsh completion: %v", err)
	}

	_, err = buf.Write([]byte(zshFinalize))
	if err != nil {
		return fmt.Errorf("error while generating zsh completion: %v", err)
	}

	// remove the command "completion" from auto-completion
	code := buf.String()
	code = strings.Replace(code, `commands+=("completion")`, "", -1)

	fmt.Print(code)
	return nil
}
