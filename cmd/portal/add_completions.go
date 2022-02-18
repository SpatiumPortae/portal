package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"www.github.com/ZinoKader/portal/tools"
)

var addCompletionsCmd = &cobra.Command{
	Use:   "add-completions",
	Short: "Adds shell completions for all `portal` subcommands",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		shellBinPath := os.Getenv("SHELL")
		if len(shellBinPath) == 0 {
			fmt.Printf("Completions not added - could not find which shell is used.\nTo add completions manually, add the following to your config:\n\n%s", SHELL_COMPLETION_SCRIPT)
			os.Exit(1)
		}

		shellPathComponents := strings.Split(os.Getenv("SHELL"), "/")
		usedShell := shellPathComponents[len(shellPathComponents)-1]
		if !tools.Contains([]string{"bash", "zsh"}, usedShell) {
			fmt.Printf("Unsupported shell \"%s\" at path: \"%s\".", usedShell, shellBinPath)
			os.Exit(1)
		}

		err := writeShellCompletionScript(usedShell)
		if err != nil {
			fmt.Printf("Failed when adding script to shell config file: %e", err)
			os.Exit(1)
		}
		fmt.Println("Successfully added completions to your shell config. Run 'source' on your shell config or restart your shell.")
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
