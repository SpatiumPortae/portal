package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"www.github.com/ZinoKader/portal/tools"
)

// rootCmd is the top level `portal` command on which the other subcommands are attached to.
var rootCmd = &cobra.Command{
	Use:   "portal",
	Short: "Portal is a quick and easy command-line file transfer utility from any computer to another.",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))
	},
}

// Entry point of the application.
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Initialization of cobra and viper.
func init() {
	// tools.RandomSeed()

	cobra.OnInitialize(initViperConfig)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Specifes if portal logs debug information to a file on the format `.portal-[command].log` In the current directory")
	// Setup viper config.
	// Add cobra subcommands.
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(addCompletionsCmd)
}

// HELPER FUNCTIONS

// initViperConfig initializes the viper config.
// It creates a `.portal.yml` file at the home directory if it has not been created earlier
// NOTE: The precedence levels of viper are the following: flags -> config file -> defaults
// See https://github.com/spf13/viper#why-viper
func initViperConfig() {
	// Set default values
	viper.SetDefault("verbose", false)
	viper.SetDefault("rendezvousPort", DEFAULT_RENDEZVOUS_PORT)
	viper.SetDefault("rendezvousAddress", DEFAULT_RENDEZVOUS_ADDRESS)

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
		//NOTE: perhaps should be an empty file initially, as we would not want defaut IP to be written to a file on the user host
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
	err := tools.ValidateHostname(viper.GetString("rendezvousAddress"))
	// neither a valid IP nor a valid hostname was provided
	if (rendezvouzAdress == nil) && err != nil {
		return errors.New("Invalid IP or hostname provided.")
	}
	return nil
}

func setupLoggingFromViper(cmd string) error {
	if viper.GetBool("verbose") {
		f, err := tea.LogToFile(fmt.Sprintf(".portal-%s.log", cmd), fmt.Sprintf("portal-%s: \n", cmd))
		if err != nil {
			return fmt.Errorf("Could not log to the provided file.\n")
		}
		defer f.Close()
	} else {
		log.SetOutput(io.Discard)
	}
	return nil
}
