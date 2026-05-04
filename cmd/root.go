package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "GraphQL-Curl",
	Short: "GraphQL-Curl is a CLI tool to convert GraphQL queries to cURL commands.",
	Long:  `GraphQL-Curl is a CLI tool that takes GraphQL queries as input and generates corresponding cURL commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to GraphQL-Curl! Use --help to see available commands.")

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
