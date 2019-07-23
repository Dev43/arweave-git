package cmd

import (
	"log"

	"github.com/Dev43/arweave-go/transactor"
	"github.com/spf13/cobra"
)

// Global variable for the package
var ar *transactor.Transactor
var url string

var rootCmd = &cobra.Command{
	Use:   "git",
	Short: "Arweave Git Portal",
	Long:  `Send Git material to the Arweave network`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// check the url passed
		// check if we can connect
		var err error
		ar, err = transactor.NewTransactor(url)
		if err != nil {
			panic(err)
		}
		_ = ar

	},
}

// Execute is our top line function for all CLI commands
func Execute() {
	rootCmd.PersistentFlags().StringVar(&url, "url", "127.0.0.1", "URL of the arweave host")
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
