package cmd

import (
	"github.com/spf13/cobra"
)

var recordsCmd = &cobra.Command{
	Use:   "records",
	Short: "Manage records",
	Long: `Commands for managing password records.
Requires authentication via 'gophkeeper login' first.`,
}

func init() {
	rootCmd.AddCommand(recordsCmd)
}
