package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SpatiumPortae/portal/internal/config"
	"github.com/alecthomas/chroma/quick"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configResetCmd)
}

var configCmd = &cobra.Command{
	Use:       "config",
	Short:     "View and configure options",
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{configPathCmd.Name(), configViewCmd.Name(), configEditCmd.Name(), configResetCmd.Name()},
	Run:       func(cmd *cobra.Command, args []string) {},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Output the path of the config file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(viper.ConfigFileUsed())
	},
}

var configViewCmd = &cobra.Command{
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

var configEditCmd = &cobra.Command{
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

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset to the default configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := viper.ConfigFileUsed()
		err := os.WriteFile(configPath, config.ToYaml(config.GetDefault()), 0)
		if err != nil {
			return fmt.Errorf("config file (%s) could not be read/written to: %w", configPath, err)
		}
		return nil
	},
}

// -------------------------------------------------- Helper Functions -------------------------------------------------

// initConfig initializes the viper config.
// `config.yml` is created in $HOME/.config/portal if not already existing.
// NOTE: The precedence levels of viper are the following: flags -> config file -> defaults.
func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	configPath := filepath.Join(home, config.CONFIGS_DIR_NAME, config.PORTAL_CONFIG_DIR_NAME)
	viper.AddConfigPath(configPath)
	viper.SetConfigName(config.CONFIG_FILE_NAME)
	viper.SetConfigType(config.CONFIG_FILE_EXT)

	if err := viper.ReadInConfig(); err != nil {
		// Create config file if not found.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			err := os.MkdirAll(configPath, os.ModePerm)
			if err != nil {
				fmt.Println("Could not create config directory:", err)
				os.Exit(1)
			}

			configFile, err := os.Create(filepath.Join(configPath, fmt.Sprintf("%s.%s", config.CONFIG_FILE_NAME, config.CONFIG_FILE_EXT)))
			if err != nil {
				fmt.Println("Could not create config file:", err)
				os.Exit(1)
			}
			defer configFile.Close()

			_, err = configFile.Write(config.ToYaml(config.GetDefault()))
			if err != nil {
				fmt.Println("Could not write defaults to config file:", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Could not read config file:", err)
			os.Exit(1)
		}
	}
}

// Sets default viper values.
func setDefaults() {
	for k, v := range config.ToMap(config.GetDefault()) {
		viper.SetDefault(k, v)
	}
}
