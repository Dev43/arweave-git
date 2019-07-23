package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	releaseCmd.AddCommand(pullCmd)
	releaseCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(releaseCmd)
}

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release commands",
	Long:  `Arweave Git release commands`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("No arguments given, exitingPread")
	},
}
