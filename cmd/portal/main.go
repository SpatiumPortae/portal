package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version represents the version of portal.
// injected at link time using -ldflags.
var version string

// Initialization of cobra and viper.
func init() {
	cobra.OnInitialize(initViperConfig)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Log debug information to a file on the format `.portal-[command].log` in the current directory")
	// Setup viper config.
	// Add cobra subcommands.
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
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
	Use: "version",
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

// initViperConfig initializes the viper config.
// It creates a `.portal.yml` file at the home directory if it has not been created earlier
// NOTE: The precedence levels of viper are the following: flags -> config file -> defaults
// See https://github.com/spf13/viper#why-viper
func initViperConfig() {
	// Set default values
	viper.SetDefault("verbose", false)
	viper.SetDefault("relay", fmt.Sprintf("%s:%d", DEFAULT_RENDEZVOUS_ADDRESS, DEFAULT_RENDEZVOUS_PORT))

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Search for config in home directory.
	viper.AddConfigPath(home)
	viper.SetConfigName(CONFIG_FILE_NAME)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		// Create config file if not found
		// NOTE: perhaps should be an empty file initially, as we would not want default IP to be written to a file on the user host
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configPath := filepath.Join(home, CONFIG_FILE_NAME)
			configFile, err := os.Create(configPath)
			if err != nil {
				fmt.Println("Could not create config file:", err)
				os.Exit(1)
			}
			defer configFile.Close()
			_, err = configFile.Write([]byte(DEFAULT_CONFIG_YAML))
			if err != nil {
				fmt.Println("Could not write defaults to config file:", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Could not read config file:", err)
			os.Exit(1)
		}
	}
}

// validateRendezvousAddressInViper validates that the `rendezvousAddress` value in viper is a valid hostname or IP
func validateRendezvousAddressInViper() error {
	rendezvouzAdress := net.ParseIP(viper.GetString("rendezvousAddress"))
	err := validateHostname(viper.GetString("rendezvousAddress"))
	// neither a valid IP nor a valid hostname was provided
	if (rendezvouzAdress == nil) && err != nil {
		return errors.New("invalid IP or hostname provided")
	}
	return nil
}

func setupLoggingFromViper(cmd string) (*os.File, error) {
	if viper.GetBool("verbose") {
		f, err := tea.LogToFile(fmt.Sprintf(".portal-%s.log", cmd), fmt.Sprintf("portal-%s: \n", cmd))
		if err != nil {
			return nil, fmt.Errorf("could not log to the provided file")
		}
		return f, nil
	}
	log.SetOutput(io.Discard)
	return nil, nil
}

// validateHostname returns an error if the domain name is not valid
// See https://tools.ietf.org/html/rfc1034#section-3.5 and
// https://tools.ietf.org/html/rfc1123#section-2.
// source: https://gist.github.com/chmike/d4126a3247a6d9a70922fc0e8b4f4013
func validateHostname(name string) error {
	switch {
	case len(name) == 0:
		return nil
	case len(name) > 255:
		return fmt.Errorf("name length is %d, can't exceed 255", len(name))
	}
	var l int
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == '.' {
			// check domain labels validity
			switch {
			case i == l:
				return fmt.Errorf("invalid character '%c' at offset %d: label can't begin with a period", b, i)
			case i-l > 63:
				return fmt.Errorf("byte length of label '%s' is %d, can't exceed 63", name[l:i], i-l)
			case name[l] == '-':
				return fmt.Errorf("label '%s' at offset %d begins with a hyphen", name[l:i], l)
			case name[i-1] == '-':
				return fmt.Errorf("label '%s' at offset %d ends with a hyphen", name[l:i], l)
			}
			l = i + 1
			continue
		}
		// test label character validity, note: tests are ordered by decreasing validity frequency
		if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b >= 'A' && b <= 'Z') {
			// show the printable unicode character starting at byte offset i
			c, _ := utf8.DecodeRuneInString(name[i:])
			if c == utf8.RuneError {
				return fmt.Errorf("invalid rune at offset %d", i)
			}
			return fmt.Errorf("invalid character '%c' at offset %d", c, i)
		}
	}
	// check top level domain validity
	switch {
	case l == len(name):
		return fmt.Errorf("missing top level domain, domain can't end with a period")
	case len(name)-l > 63:
		return fmt.Errorf("byte length of top level domain '%s' is %d, can't exceed 63", name[l:], len(name)-l)
	case name[l] == '-':
		return fmt.Errorf("top level domain '%s' at offset %d begins with a hyphen", name[l:], l)
	case name[len(name)-1] == '-':
		return fmt.Errorf("top level domain '%s' at offset %d ends with a hyphen", name[l:], l)
	case name[l] >= '0' && name[l] <= '9':
		return fmt.Errorf("top level domain '%s' at offset %d begins with a digit", name[l:], l)
	}
	return nil
}
