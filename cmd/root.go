package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "vanished-rooms",
	Short: "Chat room focused on privacy",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
