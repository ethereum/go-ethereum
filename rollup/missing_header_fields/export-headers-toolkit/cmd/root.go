package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "export-headers-toolkit",
	Short: "A toolkit for exporting and transforming missing block header fields of Scroll",
	Long: `A toolkit for exporting and transforming missing block header fields of Scroll.

The fields difficulty and extraData are missing from header data stored on L1 before EuclidV2. 
This toolkit provides commands to export the missing fields, deduplicate the data and create a 
file with the missing fields that can be used to reconstruct the correct block hashes when only reading 
data from L1.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
