package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/SpatiumPortae/portal/cmd/portal/config"
	"github.com/alecthomas/chroma/quick"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Config() *cobra.Command {

	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Output the path of the config file",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(viper.ConfigFileUsed())
		},
	}

	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "View the configured options",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := viper.ConfigFileUsed()
			config, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("config file (%s) could not be read: %w", configPath, err)
			}
			if err := quick.Highlight(os.Stdout, string(config), "yaml", "terminal256", "onedark"); err != nil {
				// Failed to highlight output, output un-highlighted config file contents.
				fmt.Println(string(config))
			}
			return nil
		},
	}

	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := viper.ConfigFileUsed()
			// Strip arguments from editor variable -- allows exec.Command to lookup the editor executable correctly.
			editor, _, _ := strings.Cut(os.Getenv("EDITOR"), " ")
			if len(editor) == 0 {
				//lint:ignore ST1005 error string is command output
				return fmt.Errorf(
					"Could not find default editor (is the $EDITOR variable set?)\nOptionally you can open the file (%s) manually", configPath,
				)
			}

			editorCmd := exec.Command(editor, configPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("failed to open file (%s) in editor (%s): %w", configPath, editor, err)
			}
			return nil
		},
	}
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset to the default configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := viper.ConfigFileUsed()
			err := os.WriteFile(configPath, config.GetDefault().Yaml(), 0)
			if err != nil {
				return fmt.Errorf("config file (%s) could not be read/written to: %w", configPath, err)
			}
			return nil
		},
	}
	configCmd := &cobra.Command{
		Use:       "config",
		Short:     "View and configure options",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{pathCmd.Name(), viewCmd.Name(), editCmd.Name(), resetCmd.Name()},
		Run:       func(cmd *cobra.Command, args []string) {},
	}

	configCmd.AddCommand(pathCmd)
	configCmd.AddCommand(viewCmd)
	configCmd.AddCommand(editCmd)
	configCmd.AddCommand(resetCmd)

	return configCmd
}
