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

// Set flags.
func init() {
	// Add subcommand flags (dummy default values as default values are handled through viper)
	desc := `Address of relay server. Accepted formats:
  - 127.0.0.1:8080
  - [::1]:8080
  - somedomain.com
	`
	sendCmd.Flags().StringP("relay", "r", "", desc)
}

// ------------------------------------------------------ Command ------------------------------------------------------

// sendCmd cobra command for `portal send`.
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send one or more files",
	Long:  "The send command adds one or more files to be sent. Files are archived and compressed before sending.",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Bind flags to viper
		if err := viper.BindPFlag("relay", cmd.Flags().Lookup("relay")); err != nil {
			return fmt.Errorf("binding relay flag: %w", err)
		}
		return nil

	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sender.Init(); err != nil {
			return err
		}
		file.RemoveTemporaryFiles(file.SEND_TEMP_FILE_NAME_PREFIX)

		relayAddr := viper.GetString("relay")
		if err := validateAddress(relayAddr); err != nil {
			return fmt.Errorf("%w: (%s) is not a valid relay address", err, relayAddr)
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

// ------------------------------------------------------ Handler ------------------------------------------------------

// handleSendCommand is the sender application.
func handleSendCommand(fileNames []string) {
	var opts []senderui.Option
	ver, err := semver.Parse(version)
	// Conditionally add option to sender ui
	if err == nil {
		opts = append(opts, senderui.WithVersion(ver))
	}
	relayAddr := viper.GetString("relay")
	sender := senderui.New(fileNames, relayAddr, opts...)
	if _, err := sender.Run(); err != nil {
		fmt.Println("Error initializing UI", err)
		os.Exit(1)
	}
	fmt.Println("")
	os.Exit(0)
}
