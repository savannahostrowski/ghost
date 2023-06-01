package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v52/github"
	"github.com/savannahostrowski/ghost/ai"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the ghost CLI",
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Println("OPENAI_API_KEY environment variable not set")
			os.Exit(1)
		}
		client := github.NewClient(nil)

		var files []string
		if repo, _ := cmd.Flags().GetString("repo"); repo != "" {
		// Get a list of files in the repo
			owner, repoName := getOwnerAndRepoName(repo)
			files = getFilesInRepo(client, owner, repoName)

		} else {
			files = getFilesInCurrentDirAndSubDirs()
		}
		ai.ChatGPTRequest(files)

	},
}

func getOwnerAndRepoName(repoUrl string)  (string, string) {
	// Get owner and reponame from uRL
	repoUrlList := strings.Split(repoUrl, "/")
	owner := repoUrlList[len(repoUrlList)-2]
	repoName := repoUrlList[len(repoUrlList)-1]
	return owner, repoName
}

func getFilesInRepo(client *github.Client, owner string, repoName string) []string {
	// Get a list of files in the repo
	context := context.Background()
	_, dirContent, _, err := client.Repositories.GetContents(context, owner, repoName, "", nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	files := []string{}
	for _, file := range dirContent {
		if *file.Type == "file" {
			files = append(files, *file.Path)
		}
	}
	return files
}

func getFilesInCurrentDirAndSubDirs() []string {
	files := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if path[0] == '.' {
			return nil
		}

		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return files
}

