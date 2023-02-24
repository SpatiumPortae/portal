package main

import (
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version represents the version of portal.
// injected at link time using -ldflags.
var version string

// Initialization of cobra and viper.
func init() {
	initConfig()
	setDefaults()

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Log debug information to a file on the format `.portal-[command].log` in the current directory")
	// Add cobra subcommands.
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
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
	Use:   "version",
	Short: "Display the installed version of portal",
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

func setupLoggingFromViper(cmd string) (*os.File, error) {
	if viper.GetBool("verbose") {
		f, err := tea.LogToFile(fmt.Sprintf(".portal-%s.log", cmd), fmt.Sprintf("portal-%s: \n", cmd))
		if err != nil {
			return nil, fmt.Errorf("could not log to the provided file: %w", err)
		}
		return f, nil
	}
	log.SetOutput(io.Discard)
	return nil, nil
}
