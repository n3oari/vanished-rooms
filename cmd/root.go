package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "vanished-rooms",
	Short: "Chat enfocado en privacidad",
}

// Execute añade todos los comandos hijos al comando raíz y lo ejecuta.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
