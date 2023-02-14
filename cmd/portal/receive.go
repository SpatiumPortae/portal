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

// receiveCmd is the cobra command for `portal receive`
var receiveCmd = &cobra.Command{
	Use:   "receive",
	Short: "Receive files",
	Long:  "The receive command receives files from the sender with the matching password.",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cobra.CompDebug(toComplete, true)
		split := strings.Split(toComplete, "-")
		if len(split) > 4 || len(split) == 0 {
			return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		if len(split) == 1 {
			if _, err := strconv.Atoi(split[0]); err != nil {
				return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}
			return []string{fmt.Sprintf("%s-", split[0])}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		suggs := filterPrefix(removeElems(data.SpaceWordList, split[:len(split)-1]), split[len(split)-1])
		var res []string
		for _, sugg := range suggs {
			components := append(split[:len(split)-1], sugg)
			password := strings.Join(components, "-")
			if len(components) < 4 {
				password += "-"
			}
			res = append(res, password)
		}
		return res, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind flags to viper
		//nolint
		viper.BindPFlag("rendezvousPort", cmd.Flags().Lookup("rendezvous-port"))
		//nolint
		viper.BindPFlag("rendezvousAddress", cmd.Flags().Lookup("rendezvous-address"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		file.RemoveTemporaryFiles(file.RECEIVE_TEMP_FILE_NAME_PREFIX)
		err := validateRendezvousAddressInViper()
		if err != nil {
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

// Setup flags
func init() {
	// Add subcommand flags (dummy default values as default values are handled through viper)
	//TODO: recactor this into a single flag for providing a TCPAddr
	receiveCmd.Flags().IntP("rendezvous-port", "p", 0, "port on which the rendezvous server is running")
	receiveCmd.Flags().StringP("rendezvous-address", "a", "", "host address for the rendezvous server")
}

// handleReceiveCommand is the receive application.
func handleReceiveCommand(password string) {
	addr := viper.GetString("rendezvousAddress")
	port := viper.GetInt("rendezvousPort")
	var opts []receiver.Option
	ver, err := semver.Parse(version)
	if err == nil {
		opts = append(opts, receiver.WithVersion(ver))
	}
	receiver := receiver.New(fmt.Sprintf("%s:%d", addr, port), password, opts...)

	if _, err := receiver.Run(); err != nil {
		fmt.Println("Error initializing UI", err)
		os.Exit(1)
	}
	fmt.Println("")
	os.Exit(0)
}

// ------------------------------------------------ Password Completion ------------------------------------------------

func completionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	split := strings.Split(toComplete, "-")

	if len(split) > 4 || len(split) == 0 {
		return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}
	if len(split) == 1 {
		if _, err := strconv.Atoi(split[0]); err != nil {
			return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		return []string{fmt.Sprintf("%s-", split[0])}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}
	// Remove previous components of password, and filter based on prefix
	suggs := filterPrefix(removeElems(data.SpaceWordList, split[:len(split)-1]), split[len(split)-1])
	var res []string
	for _, sugg := range suggs {
		components := append(split[:len(split)-1], sugg)
		password := strings.Join(components, "-")
		if len(components) < 4 {
			password += "-"
		}
		res = append(res, password)
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
