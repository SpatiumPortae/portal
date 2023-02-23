package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/SpatiumPortae/portal/data"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/password"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/ui/receiver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

// Setup flags.
func init() {
	// Add subcommand flags (dummy default values as default values are handled through viper).
	receiveCmd.Flags().StringP("relay", "r", "", "address of the relay server")
}

// ------------------------------------------------------ Command ------------------------------------------------------

// receiveCmd is the cobra command for `portal receive`
var receiveCmd = &cobra.Command{
	Use:               "receive",
	Short:             "Receive files",
	Long:              "The receive command receives files from the sender with the matching password.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: passwordCompletion,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// BindvalidateRelayInViper
		if err := viper.BindPFlag("relay", cmd.Flags().Lookup("relay")); err != nil {
			return fmt.Errorf("binding relay flag: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		file.RemoveTemporaryFiles(file.RECEIVE_TEMP_FILE_NAME_PREFIX)
		if err := validateRelayInViper(); err != nil {
			return err
		}
		logFile, err := setupLoggingFromViper("receive")
		if err != nil {
			return err
		}
		defer logFile.Close()
		pwd := args[0]
		if !password.IsValid(pwd) {
			return fmt.Errorf("invalid password format")
		}
		handleReceiveCommand(pwd)
		return nil
	},
}

// ------------------------------------------------------ Handler ------------------------------------------------------

// handleReceiveCommand is the receive application.
func handleReceiveCommand(password string) {
	var opts []receiver.Option
	ver, err := semver.Parse(version)
	if err == nil {
		opts = append(opts, receiver.WithVersion(ver))
	}
	receiver := receiver.New(viper.GetString("relay"), password, opts...)

	if _, err := receiver.Run(); err != nil {
		fmt.Println("Error initializing UI", err)
		os.Exit(1)
	}
	fmt.Println("")
	os.Exit(0)
}

// ------------------------------------------------ Password Completion ------------------------------------------------

func passwordCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	components := strings.Split(toComplete, "-")

	if len(components) > password.Length+1 || len(components) == 0 {
		return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}
	if len(components) == 1 {
		if _, err := strconv.Atoi(components[0]); err != nil {
			return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		return []string{fmt.Sprintf("%s-", components[0])}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}
	// Remove previous components of password, and filter based on prefix.
	suggs := filterPrefix(removeElems(data.SpaceWordList, components[:len(components)-1]), components[len(components)-1])
	var res []string
	for _, sugg := range suggs {
		components := append(components[:len(components)-1], sugg)
		pw := strings.Join(components, "-")
		if len(components) <= password.Length {
			pw += "-"
		}
		res = append(res, pw)
	}
	return res, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func removeElems(src []string, elems []string) []string {
	var res []string
	for _, elem := range src {
		if slices.Contains(elems, elem) {
			continue
		}
		res = append(res, elem)
	}
	return res
}

func filterPrefix(src []string, prefix string) []string {
	var res []string
	for _, elem := range src {
		if strings.HasPrefix(elem, prefix) {
			res = append(res, elem)
		}
	}
	return res
}
