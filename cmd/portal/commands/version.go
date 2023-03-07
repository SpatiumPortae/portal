package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Version(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the installed version of portal",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
			os.Exit(0)
		},
	}

}
