package main

import (
	"github.com/spf13/cobra"
	"www.github.com/ZinoKader/portal/pkg/rendezvous"
)

// serveCmd is the cobra command for `portal serve`
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the rendezvous-server",
	Long:  "The serve command serves the rendezvous-server locally.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		server := rendezvous.NewServer(port)
		server.Start()
	},
}

// Add `port` flag.
// NOTE:: The `port` flag is required and not managed through viper.
func init() {
	serveCmd.Flags().IntP("port", "p", 0, "Port to run the portal rendezvous server on")
	serveCmd.MarkFlagRequired("port")
}
