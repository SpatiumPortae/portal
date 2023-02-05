package main

import (
	"fmt"
	"os"

	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/internal/sender"
	senderui "github.com/SpatiumPortae/portal/ui/sender"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// sendCmd cobra command for `portal send`.
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send one or more files",
	Long:  "The send command adds one or more files to be sent. Files are archived and compressed before sending.",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind flags to viper
		//nolint:errcheck
		viper.BindPFlag("rendezvousPort", cmd.Flags().Lookup("rendezvous-port"))
		//nolint:errcheck
		viper.BindPFlag("rendezvousAddress", cmd.Flags().Lookup("rendezvous-address"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sender.Init(); err != nil {
			return err
		}
		file.RemoveTemporaryFiles(file.SEND_TEMP_FILE_NAME_PREFIX)
		if err := validateRendezvousAddressInViper(); err != nil {
			return err
		}

		logFile, err := setupLoggingFromViper("send")
		if err != nil {
			return err
		}
		defer logFile.Close()

		handleSendCommand(args)
		return nil
	},
}

// Set flags.
func init() {
	// Add subcommand flags (dummy default values as default values are handled through viper)
	//TODO: refactor into a single flag providing a string
	sendCmd.Flags().IntP("rendezvous-port", "p", 0, "port on which the rendezvous server is running")
	sendCmd.Flags().StringP("rendezvous-address", "a", "", "host address for the rendezvous server")
}

// handleSendCommand is the sender application.
func handleSendCommand(fileNames []string) {
	addr := viper.GetString("rendezvousAddress")
	port := viper.GetInt("rendezvousPort")
	var opts []senderui.Option
	ver, err := semver.Parse(version)
	// Conditionally add option to sender ui
	if err == nil {
		opts = append(opts, senderui.WithVersion(ver))
	}
	sender := senderui.New(fileNames, fmt.Sprintf("%s:%d", addr, port), opts...)
	if err := sender.Start(); err != nil {
		fmt.Println("Error initializing UI", err)
		os.Exit(1)
	}
	fmt.Println("")
	os.Exit(0)
}
