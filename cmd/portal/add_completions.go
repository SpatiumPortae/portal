package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"www.github.com/ZinoKader/portal/tools"
)

// SHELL_COMPLETION_SCRIPT is the completion script that will be added to the shell rc file.
const SHELL_COMPLETION_SCRIPT = `_portal_completions() {
	args=("${COMP_WORDS[@]:1:$COMP_CWORD}")

	local IFS=$'\n'
	COMPREPLY=($(GO_FLAGS_COMPLETION=1 ${COMP_WORDS[0]} "${args[@]}"))
	return 1
}
complete -F _portal_completions portal
`

// addCompletionsCmd is the cobra command for `portal add-completions`
var addCompletionsCmd = &cobra.Command{
	Use:   "add-completions",
	Short: "Adds shell completions for all `portal` subcommands",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		shellBinPath := os.Getenv("SHELL")
		if len(shellBinPath) == 0 {
			return fmt.Errorf("Completions not added - could not find which shell is used.\nTo add completions manually, add the following to your config:\n\n%s", SHELL_COMPLETION_SCRIPT)
		}

		shellPathComponents := strings.Split(os.Getenv("SHELL"), "/")
		usedShell := shellPathComponents[len(shellPathComponents)-1]
		if !tools.Contains([]string{"bash", "zsh"}, usedShell) {
			return fmt.Errorf("Unsupported shell \"%s\" at path: \"%s\".", usedShell, shellBinPath)
		}

		err := writeShellCompletionScript(usedShell)
		if err != nil {
			return fmt.Errorf("Failed when adding script to shell config file: %e", err)
		}
		fmt.Println("Successfully added completions to your shell config. Run 'source' on your shell config or restart your shell.")
		return nil
	},
}

// writeShellCompletionScript writes the completion script to the specified shell name
func writeShellCompletionScript(shellName string) error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	shellConfigName := fmt.Sprintf(".%src", shellName)
	shellConfigPath := path.Join(homedir, shellConfigName)
	f, err := os.OpenFile(shellConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("\n# portal shell completion\n%s\n", SHELL_COMPLETION_SCRIPT)); err != nil {
		return err
	}

	return nil
}
