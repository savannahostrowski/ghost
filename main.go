package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// func initialModel(prompt, apiKey string) tea.Model {
// 	return model{
// 		prompt: prompt,
// 		apiKey: apiKey,
// 	}

// 	model := model{
// 		apiKey: apiKey,
// 		input: input,
// 		spinner: spinny,


// }

// func (m model) Init() tea.Cmd {
// 	return nil
// }

// func (m model) Update() (tea.Model, tea.Cmd) {
// 	return m, nil
// }

// func (m model) View() string {
// 	return fmt.Sprintf("Hello")
// }

var rootCmd = &cobra.Command{
	Use:   "ghost",
	Short: "A CLI that uses AI to help scaffold a GitHub Actions workflow for your codebase",
	Run: func(cmd *cobra.Command, args []string) {
		// var prompt string

		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Println("OPENAI_API_KEY environment variable not set")
			os.Exit(1)
		}

		// p := tea.NewProgram(initialModel(prompt, apiKey))
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
