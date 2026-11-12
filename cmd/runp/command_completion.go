package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

// completionShells lists the shells "runp completion" can generate a script for.
var completionShells = []string{"bash", "zsh", "fish"}

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
	Action:       doCompletion,
	BashComplete: completionWords(completionShells),
}

// completingFlag reports whether the shell is completing a flag: the scripts
// pass a lone "-" (or the partial flag) as the last argument in that case.
func completingFlag(c *cli.Context) bool {
	n := c.NArg()
	return n > 0 && strings.HasPrefix(c.Args().Get(n-1), "-")
}

// withFlagCompletion wraps a BashComplete func so that flags are still
// suggested when the word being completed starts with "-".
func withFlagCompletion(fn cli.BashCompleteFunc) cli.BashCompleteFunc {
	return func(c *cli.Context) {
		if completingFlag(c) {
			cli.DefaultCompleteWithFlags(c.Command)(c)
			return
		}
		fn(c)
	}
}

// completionWords returns a BashComplete func that suggests words for the
// first positional argument and nothing afterwards.
func completionWords(words []string) cli.BashCompleteFunc {
	return withFlagCompletion(func(c *cli.Context) {
		if c.NArg() > 0 {
			return
		}
		for _, w := range words {
			fmt.Println(w)
		}
	})
}

// completionSettingKeys suggests the supported settings keys (runp config get/set).
func completionSettingKeys(c *cli.Context) {
	keys := make([]string, 0, len(core.SupportedSettingKeys()))
	for k := range core.SupportedSettingKeys() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	completionWords(keys)(c)
}

// completionUnitNames is a BashComplete func for commands that accept a unit name.
// It prints each unit name on its own line when --generate-bash-completion is active.
// It never writes anything else to stdout: whatever it prints becomes a candidate.
var completionUnitNames = withFlagCompletion(completeUnitNames)

func completeUnitNames(c *cli.Context) {
	if c.NArg() > 0 {
		return
	}
	// app.Before does not run in completion mode, so the logger used by
	// lib/core is not configured yet: install a quiet one.
	if ui == nil {
		ui = core.CreateMainLoggerWithLevel(" ", 6, "%s> ", core.LogLevelError, false)
		core.ConfigureUI(ui, core.LoggerConfig{Level: core.LogLevelError, Color: false})
	}
	runpfilePath := resolveRunpfileArg(c)
	if isURL(runpfilePath) {
		// Never hit the network to complete a word.
		return
	}
	resolved, err := core.ResolveRunpfilePath(runpfilePath)
	if err != nil {
		return
	}
	rf, err := core.LoadRunpfileFromPath(resolved)
	if err != nil || rf == nil {
		return
	}
	names := make([]string, 0, len(rf.Units))
	for name := range rf.Units {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Println(n)
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

// The bash and zsh scripts delegate to runp itself (--generate-bash-completion),
// so subcommands, flags and unit names always match the installed binary.
//
// When the word being completed starts with "-", runp is asked for flags by
// passing a lone "-" rather than the word itself: passing "--" would end flag
// parsing and make runp execute the command instead of completing it.

const bashCompletion = `# runp bash completion
# Source this file or add to ~/.bashrc:
#   source <(runp completion bash)
_runp_bash_completion() {
    local cur opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == -* ]]; then
        opts=$("${COMP_WORDS[@]:0:$COMP_CWORD}" - --generate-bash-completion 2>/dev/null)
    else
        opts=$("${COMP_WORDS[@]:0:$COMP_CWORD}" --generate-bash-completion 2>/dev/null)
    fi
    COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
    return 0
}
complete -o bashdefault -o default -F _runp_bash_completion runp
`

const zshCompletion = `#compdef runp
# runp zsh completion
# Source this file or add to ~/.zshrc (after compinit):
#   source <(runp completion zsh)
_runp_zsh_completion() {
    local -a opts
    local cur
    cur=${words[-1]}
    if [[ "$cur" == -* ]]; then
        opts=("${(@f)$(${words[@]:0:#words[@]-1} - --generate-bash-completion 2>/dev/null)}")
    else
        opts=("${(@f)$(${words[@]:0:#words[@]-1} --generate-bash-completion 2>/dev/null)}")
    fi
    if [[ "${opts[1]}" != "" ]]; then
        _describe 'values' opts
    else
        _files
    fi
}
compdef _runp_zsh_completion runp
`

// The fish script is written by hand to get per-item descriptions; argument
// values (unit names, setting keys, shells) still come from runp itself.
// TestFishCompletionCoversCommands keeps it in sync with the command definitions.
const fishCompletion = `# runp fish completion
# Save to ~/.config/fish/completions/runp.fish:
#   runp completion fish > ~/.config/fish/completions/runp.fish

function __fish_runp_no_subcommand
    for word in (commandline -opc)[2..-1]
        if contains -- $word up list ls status ps validate reload encrypt config completion help h
            return 1
        end
    end
    return 0
end

function __fish_runp_using_subcommand
    for word in (commandline -opc)[2..-1]
        if contains -- $word $argv
            return 0
        end
    end
    return 1
end

# Ask runp for argument candidates (unit names, setting keys, shells),
# honouring the flags already typed (e.g. --file).
function __fish_runp_args
    set -l tokens (commandline -opc)
    $tokens --generate-bash-completion 2>/dev/null
end

# Global flags
complete -c runp -n '__fish_runp_no_subcommand' -s l -l log-level -x -a 'trace debug info warn error' -d 'Log verbosity'
complete -c runp -n '__fish_runp_no_subcommand' -s d -l debug -d 'Deprecated: use --log-level debug'
complete -c runp -n '__fish_runp_no_subcommand' -s q -l quiet -d 'Deprecated: use --log-level warn'
complete -c runp -n '__fish_runp_no_subcommand' -s C -l no-color -d 'Disable colored output'
complete -c runp -n '__fish_runp_no_subcommand' -s f -l file -r -F -d 'Path to Runpfile'
complete -c runp -n '__fish_runp_no_subcommand' -s n -l dry-run -d 'Print what would be executed'
complete -c runp -n '__fish_runp_no_subcommand' -s h -l help -d 'Show help'
complete -c runp -n '__fish_runp_no_subcommand' -s v -l version -d 'Print the version'

# Subcommands
complete -c runp -f -n '__fish_runp_no_subcommand' -a up -d 'Start all processes defined in the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a 'list ls' -d 'List all units defined in the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a 'status ps' -d 'Show runtime status of all units'
complete -c runp -f -n '__fish_runp_no_subcommand' -a validate -d 'Validate the Runpfile'
complete -c runp -f -n '__fish_runp_no_subcommand' -a reload -d 'Stop and restart a single unit'
complete -c runp -f -n '__fish_runp_no_subcommand' -a encrypt -d 'Encrypt a secret value'
complete -c runp -f -n '__fish_runp_no_subcommand' -a config -d 'Manage runp user settings'
complete -c runp -f -n '__fish_runp_no_subcommand' -a completion -d 'Print shell completion script'
complete -c runp -f -n '__fish_runp_no_subcommand' -a help -d 'Show help for a command'

# Flags shared by the commands that read a Runpfile
for cmd in up list ls status ps validate reload
    complete -c runp -n "__fish_runp_using_subcommand $cmd" -s f -l file -r -F -d 'Path to Runpfile or HTTP/HTTPS URL'
    complete -c runp -n "__fish_runp_using_subcommand $cmd" -l checksum -x -d 'Expected SHA-256 checksum (sha256:<hex>)'
end
for cmd in up list ls reload
    complete -c runp -n "__fish_runp_using_subcommand $cmd" -s V -l var -x -d 'Runtime variable (key=value)'
end
for cmd in up reload encrypt
    complete -c runp -n "__fish_runp_using_subcommand $cmd" -s k -l key -x -d 'Secret key (prefer --key-env)'
    complete -c runp -n "__fish_runp_using_subcommand $cmd" -l key-env -x -a '(set -nx)' -d 'Env var containing the secret key'
end

# up
complete -c runp -n '__fish_runp_using_subcommand up' -s n -l dry-run -d 'Print what would be executed'

# list
complete -c runp -n '__fish_runp_using_subcommand list ls' -s o -l output -x -a 'table json' -d 'Output format'

# reload: unit names
complete -c runp -f -n '__fish_runp_using_subcommand reload' -a '(__fish_runp_args)' -d 'Unit'

# config
complete -c runp -f -n '__fish_runp_using_subcommand config; and not __fish_runp_using_subcommand get set show' -a get -d 'Print value of a setting key'
complete -c runp -f -n '__fish_runp_using_subcommand config; and not __fish_runp_using_subcommand get set show' -a set -d 'Write a setting key'
complete -c runp -f -n '__fish_runp_using_subcommand config; and not __fish_runp_using_subcommand get set show' -a show -d 'Print full settings file'
complete -c runp -f -n '__fish_runp_using_subcommand config; and __fish_runp_using_subcommand get set' -a '(__fish_runp_args)' -d 'Setting key'

# completion
complete -c runp -f -n '__fish_runp_using_subcommand completion' -a '(__fish_runp_args)' -d 'Shell'
`
