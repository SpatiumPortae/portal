package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version represents the version of portal.
// injected at link time using -ldflags.
var version string

// Initialization of cobra and viper.
func init() {
	cobra.OnInitialize(initViperConfig)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Log debug information to a file on the format `.portal-[command].log` in the current directory")
	// Setup viper config.
	// Add cobra subcommands.
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}

// ------------------------------------------------------ Command ------------------------------------------------------

// rootCmd is the top level `portal` command on which the other subcommands are attached to.
var rootCmd = &cobra.Command{
	Use:   "portal",
	Short: "Portal is a quick and easy command-line file transfer utility from any computer to another.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		//nolint:errcheck
		viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	},
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
		os.Exit(0)
	},
}

// Entry point of the application.
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// -------------------------------------------------- Helper Functions -------------------------------------------------

// initViperConfig initializes the viper config.
// It creates a `.portal.yml` file at the home directory if it has not been created earlier
// NOTE: The precedence levels of viper are the following: flags -> config file -> defaults
// See https://github.com/spf13/viper#why-viper
func initViperConfig() {
	// Set default values
	viper.SetDefault("verbose", false)
	viper.SetDefault("relay", fmt.Sprintf("%s:%d", DEFAULT_RENDEZVOUS_ADDRESS, DEFAULT_RENDEZVOUS_PORT))

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Search for config in home directory.
	viper.AddConfigPath(home)
	viper.SetConfigName(CONFIG_FILE_NAME)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		// Create config file if not found
		// NOTE: perhaps should be an empty file initially, as we would not want default IP to be written to a file on the user host
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configPath := filepath.Join(home, CONFIG_FILE_NAME)
			configFile, err := os.Create(configPath)
			if err != nil {
				fmt.Println("Could not create config file:", err)
				os.Exit(1)
			}
			defer configFile.Close()
			_, err = configFile.Write([]byte(DEFAULT_CONFIG_YAML))
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

func setupLoggingFromViper(cmd string) (*os.File, error) {
	if viper.GetBool("verbose") {
		f, err := tea.LogToFile(fmt.Sprintf(".portal-%s.log", cmd), fmt.Sprintf("portal-%s: \n", cmd))
		if err != nil {
			return nil, fmt.Errorf("could not log to the provided file")
		}
		return f, nil
	}
	log.SetOutput(io.Discard)
	return nil, nil
}
