package commands

import (
	"fmt"

	"github.com/SpatiumPortae/portal/internal/rendezvous"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Serve(version string) *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the relay server",
		Long:  "The serve command serves the relay server locally.",
		Args:  cobra.MatchAll(cobra.ExactArgs(0), cobra.NoArgs),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlag("relay_serve_port", cmd.Flags().Lookup("port")); err != nil {
				return fmt.Errorf("binding relay-port flag: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ver, err := semver.Parse(version)
			if err != nil {
				return fmt.Errorf("server requires version to be set: %w", err)
			}
			server := rendezvous.NewServer(viper.GetInt("relay_serve_port"), ver)
			server.Start()
			return nil
		},
	}
	serveCmd.Flags().IntP("port", "p", 0, "port to run the portal relay server on")
	return serveCmd
}
