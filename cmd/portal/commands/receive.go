package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/SpatiumPortae/portal/cmd/portal/config"
	receiver_tui "github.com/SpatiumPortae/portal/cmd/portal/tui/receiver"
	"github.com/SpatiumPortae/portal/data"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/password"
	"github.com/SpatiumPortae/portal/internal/portal"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

// ------------------------------------------------------ Receive ------------------------------------------------------

func Receive(version string) *cobra.Command {
	receiveCmd := &cobra.Command{
		Use:               "receive",
		Short:             "Receive files",
		Long:              "The receive command receives files from the sender with the matching password.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: passwordCompletion,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags to viper.
			if err := viper.BindPFlag("relay", cmd.Flags().Lookup("relay")); err != nil {
				return fmt.Errorf("binding relay flag: %w", err)
			}

			if err := viper.BindPFlag("tui_style", cmd.Flags().Lookup("tui-style")); err != nil {
				return fmt.Errorf("binding tui-style flag: %w", err)
			}

			// Reverse the --yes/-y flag value as it has an inverse relationship
			// with the configuration value 'prompt_overwrite_files'.
			overwriteFlag := cmd.Flags().Lookup("yes")
			if overwriteFlag.Changed {
				shouldOverwrite, _ := strconv.ParseBool(overwriteFlag.Value.String())
				_ = overwriteFlag.Value.Set(strconv.FormatBool(!shouldOverwrite))
			}

			if err := viper.BindPFlag("prompt_overwrite_files", overwriteFlag); err != nil {
				return fmt.Errorf("binding yes flag: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			file.RemoveTemporaryFiles(file.RECEIVE_TEMP_FILE_NAME_PREFIX)

			relayAddr := viper.GetString("relay")
			if err := validateAddress(relayAddr); err != nil {
				return fmt.Errorf("%w: (%s) is not a valid relay address", err, relayAddr)
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
			switch viper.GetString("tui_style") {
			case config.StyleRich:
				if err := handleReceiveCommand(version, pwd); err != nil {
					return fmt.Errorf("running rich receive command: %w", err)
				}
				return nil
			case config.StyleRaw:
				if err := handleReceiveCommandRaw(version, pwd); err != nil {
					return fmt.Errorf("running raw receive command: %w", err)
				}
				return nil
			default:
				return errors.New("invalid tui style provided")
			}
		},
	}
	receiveCmd.Flags().StringP("relay", "r", "", relayFlagDesc)
	receiveCmd.Flags().BoolP("yes", "y", false, "Overwrite existing files without [Y/n] prompts")
	receiveCmd.Flags().StringP("tui-style", "s", "", tuiStyleFlagDesc)
	return receiveCmd
}

// ------------------------------------------------------ Handlers -----------------------------------------------------

// handleReceiveCommand is the receive application.
func handleReceiveCommand(version string, password string) error {
	var opts []receiver_tui.Option
	ver, err := semver.Parse(version)
	if err == nil {
		opts = append(opts, receiver_tui.WithVersion(ver))
	}
	receiver := receiver_tui.New(viper.GetString("relay"), password, opts...)

	if _, err := receiver.Run(); err != nil {
		return fmt.Errorf("running receiver tui: %w", err)
	}
	fmt.Println("")
	return nil
}

func handleReceiveCommandRaw(version string, password string) error {
	ctx := context.Background()
	relayAddr := viper.GetString("relay")
	ver, err := semver.Parse(version)
	if err != nil {
		return fmt.Errorf("parsing version: %w", err)
	}
	serverVer, err := semver.GetRendezvousVersion(ctx, relayAddr)
	if err != nil {
		return fmt.Errorf("fetching version from relay: %w", err)
	}
	if ver.Compare(serverVer) == semver.CompareOldMajor {
		return fmt.Errorf("incompatible version %s -> %s", ver, serverVer)
	}
	cnf := portal.Config{
		RendezvousAddr: relayAddr,
	}
	temp, err := os.CreateTemp(os.TempDir(), file.RECEIVE_TEMP_FILE_NAME_PREFIX)
	if err != nil {
		return fmt.Errorf("creating temp receiver file: %w", err)
	}

	if err := portal.Receive(ctx, temp, password, &cnf); err != nil {
		return fmt.Errorf("receiving files: %w", err)
	}

	if _, err := temp.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking to start of temp file: %w", err)
	}
	unpacker := file.NewUnpacker(viper.GetBool("prompt_overwrite_files"))
	defer unpacker.Close()
	defer file.RemoveTemporaryFiles(file.RECEIVE_TEMP_FILE_NAME_PREFIX)

	if err := unpacker.Init(temp); err != nil {
		return fmt.Errorf("initialising unpacker: %w", err)
	}
	input := bufio.NewReader(os.Stdin)
	for {
		commiter, err := unpacker.Unpack()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case errors.Is(err, file.ErrUnpackFileExists):
			fmt.Printf("overwrite %s? [y/n] ", commiter.FileName())
			response, err := input.ReadString('\n')
			if err != nil {
				return fmt.Errorf("unable to read input from stdin: %w", err)
			}
			switch strings.TrimSpace(response) {
			case "y", "yes", "Y", "Yes":
				// falltrough to commit.
			case "n", "no", "N", "No":
				continue
			default:
				return errors.New("invalid response to prompt")
			}
		case err != nil:
			return fmt.Errorf("unpacking file: %w", err)
		}
		if _, err := commiter.Commit(); err != nil {
			return fmt.Errorf("commiting file %s to disk: %w", commiter.FileName(), err)
		}
	}
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
