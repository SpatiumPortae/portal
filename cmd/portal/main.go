package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/tools"
)

var (
	rootCmd = &cobra.Command{
		Use:   "portal",
		Short: "Portal is a quick and easy command-line file transfer utility from any computer to another.",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
)

func init() {
	tools.RandomSeed()

	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(addCompletionsCmd)
}

func initConfig() {
	// Set default values
	viper.SetDefault("verbose", false)
	viper.SetDefault("rendezvousPort", constants.DEFAULT_RENDEZVOUS_PORT)
	viper.SetDefault("rendezvousAddress", constants.DEFAULT_RENDEZVOUS_ADDRESS)

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Search for config in home directory.
	viper.AddConfigPath(home)
	viper.SetConfigName(constants.CONFIG_FILE_NAME)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		// Create config file if not found
		//NOTE: perhaps should be an empty file initially, as we would not want defaut IP to be written to a file on the user host
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configPath := filepath.Join(home, constants.CONFIG_FILE_NAME)
			configFile, err := os.Create(configPath)
			if err != nil {
				fmt.Println("Could not create config file:", err)
				os.Exit(1)
			}
			defer configFile.Close()
			_, err = configFile.Write([]byte(constants.DEFAULT_CONFIG_YAML))
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func validateRendezvousAddress() error {
	rendezvouzAdress := net.ParseIP(viper.GetString("rendezvousAddress"))
	err := tools.ValidateHostname(viper.GetString("rendezvousAddress"))
	// neither a valid IP nor a valid hostname was provided
	if (rendezvouzAdress == nil) && err != nil {
		return errors.New("Invalid IP or hostname provided.")
	}
	return nil
}
