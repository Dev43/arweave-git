package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git",
	Short: "Arweave Git Portal",
	Long:  `Send Git material to the Arweave network`,
}

// Execute is our top line function for all CLI commands
func Execute() {

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
