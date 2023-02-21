package main

import (
	"fmt"

	"github.com/SpatiumPortae/portal/internal/rendezvous"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/spf13/cobra"
)

// serveCmd is the cobra command for `portal serve`
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the rendezvous-server",
	Long:  "The serve command serves the rendezvous-server locally.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		ver, err := semver.Parse(version)
		if err != nil {
			return fmt.Errorf("server requires version to be set: %w", err)
		}
		server := rendezvous.NewServer(port, ver)
		server.Start()
		return nil
	},
}

// Add `port` flag.
// NOTE: The `port` flag is required and not managed through viper.
func init() {
	serveCmd.Flags().IntP("port", "p", 0, "port to run the portal rendezvous server on")
	_ = serveCmd.MarkFlagRequired("port")
}
