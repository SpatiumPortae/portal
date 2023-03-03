package main

import (
	"fmt"
	"os"

	"github.com/SpatiumPortae/portal/cmd/portal/commands"
	"github.com/SpatiumPortae/portal/cmd/portal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version represents the version of portal.
// injected at link time using -ldflags.
var version string

// -------------------------------------------------------- Root -------------------------------------------------------

func Root() (*cobra.Command, error) {
	if err := config.Init(); err != nil {
		return nil, fmt.Errorf("initialising config: %w", err)
	}
	rootCmd := &cobra.Command{
		Use:   "portal",
		Short: "Portal is a quick and easy command-line file transfer utility from any computer to another.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			//nolint:errcheck
			viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
		},
	}
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Log debug information to a file on the format `.portal-[command].log` in the current directory")
	rootCmd.AddCommand(
		commands.Send(version),
		commands.Receive(version),
		commands.Serve(version),
		commands.Version(version),
		commands.Config())
	return rootCmd, nil
}

// ------------------------------------------------------- Runner ------------------------------------------------------

// Entry point of the application.
func main() {
	rootCmd, err := Root()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
