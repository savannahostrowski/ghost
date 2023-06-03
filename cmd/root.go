package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "ghost",
		Short: "A CLI that uses AI to help scaffold a GitHub Actions workflow for your codebase",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.LocalFlags().String("repo", "", "URL to a GitHub repository to use")
}