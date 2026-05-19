package main

import (
	"fmt"
	"os"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

var commandCompletion = cli.Command{
	Name:  "completion",
	Usage: "completion [bash|zsh|fish]",
	Description: `Print the shell completion script for runp to stdout. Source it in your shell
profile to enable tab completion for subcommands, flags, and unit names.

Bash — add to ~/.bashrc or ~/.bash_profile:
  source <(runp completion bash)

Zsh — add to ~/.zshrc (after compinit):
  source <(runp completion zsh)

Fish — save to the completions directory:
  runp completion fish > ~/.config/fish/completions/runp.fish`,
	UsageText: `runp completion bash
   runp completion zsh
   runp completion fish`,
	Action: doCompletion,
}

// completionUnitNames is a BashComplete func for commands that accept a unit name.
// It prints each unit name on its own line when --generate-bash-completion is active.
func completionUnitNames(c *cli.Context) {
	runpfilePath := resolveRunpfileArg(c)
	resolved, err := core.ResolveRunpfilePath(runpfilePath)
	if err != nil {
		return
	}
	rf, err := core.LoadRunpfileFromPath(resolved)
	if err != nil {
		return
	}
	for _, u := range rf.Units {
		fmt.Println(u.Name)
	}
}

func doCompletion(c *cli.Context) error {
	shell := c.Args().First()
	switch shell {
	case "bash":
		fmt.Fprint(os.Stdout, bashCompletion)
	case "zsh":
		fmt.Fprint(os.Stdout, zshCompletion)
	case "fish":
		fmt.Fprint(os.Stdout, fishCompletion)
	default:
		return exitErrorf(exitCodeArg, "Usage: runp completion [bash|zsh|fish]\nSupported shells: bash, zsh, fish")
	}
	return nil
}

const bashCompletion = `# runp bash completion
# Source this file or add to ~/.bashrc:
#   source <(runp completion bash)
PROG=runp
_runp_bash_completion() {
    if [[ "${COMP_WORDS[0]}" != "source" ]]; then
        local cur opts
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        opts=$(${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion 2>/dev/null)
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}
complete -o bashdefault -o default -o nospace -F _runp_bash_completion $PROG
unset PROG
`

const zshCompletion = `#compdef runp
# runp zsh completion
# Source this file or add to ~/.zshrc (after compinit):
#   source <(runp completion zsh)
_runp_zsh_completion() {
    local -a opts
    local cur
    cur=${words[-1]}
    if [[ "$cur" == "-"* ]]; then
        opts=("${(@f)$(${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion 2>/dev/null)}")
    else
        opts=("${(@f)$(${words[@]:0:#words[@]-1} --generate-bash-completion 2>/dev/null)}")
    fi
    if [[ "${opts[1]}" != "" ]]; then
        _describe 'values' opts
    else
        _arguments '*:file:_files'
    fi
}
compdef _runp_zsh_completion runp
`

const fishCompletion = `# runp fish completion
# Save to ~/.config/fish/completions/runp.fish:
#   runp completion fish > ~/.config/fish/completions/runp.fish

set -l runp_subcommands up list ls status ps validate reload encrypt config completion

function __fish_runp_no_subcommand
    for word in (commandline -opc)
        if contains -- $word $runp_subcommands
            return 1
        end
    end
    return 0
end

function __fish_runp_unit_names
    runp list --output json 2>/dev/null | string match -r '"name": "([^"]+)"' | string replace -r '.*"name": "([^"]+)"' '$1'
end

function __fish_runp_using_subcommand
    set -l cmd (commandline -opc)
    for i in (seq 2 (count $cmd))
        if contains -- $cmd[$i] $argv
            return 0
        end
    end
    return 1
end

# Global flags
complete -c runp -n '__fish_runp_no_subcommand' -l log-level -d 'Log verbosity (trace, debug, info, warn, error)'
complete -c runp -n '__fish_runp_no_subcommand' -l debug -d 'Deprecated: use --log-level debug'
complete -c runp -n '__fish_runp_no_subcommand' -l quiet -d 'Deprecated: use --log-level warn'
complete -c runp -n '__fish_runp_no_subcommand' -l no-color -d 'Disable colored output'
complete -c runp -n '__fish_runp_no_subcommand' -s C -d 'Disable colored output'
complete -c runp -n '__fish_runp_no_subcommand' -l file -s f -d 'Path to Runpfile'
complete -c runp -n '__fish_runp_no_subcommand' -l dry-run -s n -d 'Print what would be executed'

# Subcommands
complete -c runp -f -n '__fish_runp_no_subcommand' -a up -d 'Start all processes defined in the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a list -d 'List all units defined in the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a ls -d 'List all units defined in the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a status -d 'Show runtime status of all units'
complete -c runp -f -n '__fish_runp_no_subcommand' -a ps -d 'Show runtime status of all units'
complete -c runp -f -n '__fish_runp_no_subcommand' -a validate -d 'Validate the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a reload -d 'Stop and restart a single unit'
complete -c runp -f -n '__fish_runp_no_subcommand' -a encrypt -d 'Encrypt a secret value'
complete -c runp -f -n '__fish_runp_no_subcommand' -a config -d 'Manage runp user settings'
complete -c runp -f -n '__fish_runp_no_subcommand' -a completion -d 'Print shell completion script'

# up flags
complete -c runp -n '__fish_runp_using_subcommand up' -l file -s f -d 'Path to Runpfile'
complete -c runp -n '__fish_runp_using_subcommand up' -l dry-run -s n -d 'Print what would be executed'
complete -c runp -n '__fish_runp_using_subcommand up' -l var -s V -d 'Runtime variable (key=value)'
complete -c runp -n '__fish_runp_using_subcommand up' -l key -s k -d 'Decryption key'
complete -c runp -n '__fish_runp_using_subcommand up' -l key-env -d 'Env var containing decryption key'

# list flags
complete -c runp -n '__fish_runp_using_subcommand list' -l file -s f -d 'Path to Runpfile'
complete -c runp -n '__fish_runp_using_subcommand list' -l output -s o -d 'Output format (table, json)'
complete -c runp -n '__fish_runp_using_subcommand ls' -l file -s f -d 'Path to Runpfile'
complete -c runp -n '__fish_runp_using_subcommand ls' -l output -s o -d 'Output format (table, json)'

# status flags
complete -c runp -n '__fish_runp_using_subcommand status' -l file -s f -d 'Path to Runpfile'
complete -c runp -n '__fish_runp_using_subcommand ps' -l file -s f -d 'Path to Runpfile'

# validate flags
complete -c runp -n '__fish_runp_using_subcommand validate' -l file -s f -d 'Path to Runpfile'

# reload flags and unit names
complete -c runp -n '__fish_runp_using_subcommand reload' -l file -s f -d 'Path to Runpfile'
complete -c runp -f -n '__fish_runp_using_subcommand reload' -a '(__fish_runp_unit_names)' -d 'Unit name'

# encrypt flags
complete -c runp -n '__fish_runp_using_subcommand encrypt' -l key -s k -d 'Encryption key'
complete -c runp -n '__fish_runp_using_subcommand encrypt' -l key-env -d 'Env var containing encryption key'

# config subcommands
complete -c runp -f -n '__fish_runp_using_subcommand config' -a get -d 'Print value of a setting key'
complete -c runp -f -n '__fish_runp_using_subcommand config' -a set -d 'Write a setting key'
complete -c runp -f -n '__fish_runp_using_subcommand config' -a show -d 'Print full settings file'

# completion shells
complete -c runp -f -n '__fish_runp_using_subcommand completion' -a bash -d 'Bash completion script'
complete -c runp -f -n '__fish_runp_using_subcommand completion' -a zsh -d 'Zsh completion script'
complete -c runp -f -n '__fish_runp_using_subcommand completion' -a fish -d 'Fish completion script'
`
