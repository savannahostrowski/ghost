package main

import (
	"os"

	"github.com/savannahostrowski/ghost/cmd"
	"github.com/charmbracelet/log"

)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Error("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
