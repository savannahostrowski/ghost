package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/enescakir/emoji"
	"github.com/muesli/reflow/indent"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

type model struct {
	spinner           spinner.Model
	isLoadingResponse bool
	detectedLanguages      string
	choice            string
	quitting          bool
	err               error
}

const (
	hotPink    = lipgloss.Color("#ff69b7")
	listHeight = 14
)

var (
	gptResultStyle = lipgloss.NewStyle().Foreground(hotPink)
	itemStyle      = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(hotPink)
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the ghost CLI",
	Run: func(cmd *cobra.Command, args []string) {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(hotPink)

		m := model{
			spinner:           s,
			isLoadingResponse: false,
			choice:            "yes",
			detectedLanguages:       "",
			quitting:          false,
			err:               nil,
		}

		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			log.Fatal("Error running program: ", err)
			os.Exit(1)
		}
	},
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// User quits ghost
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			m.quitting = true
			return m, tea.Quit
		}

		// Choose yes or no to the GPT response
		if msg.String() == "up" && m.choice == "no" {
			m.choice = "yes"
		}
		if msg.String() == "down" && m.choice == "yes" {
			m.choice = "no"
		}

	default:
		var cmd tea.Cmd
		m.isLoadingResponse = true
		m.spinner, _ = m.spinner.Update(msg)

		// Get files in current directory and subdirectories for sending to the model
		files := getFilesInCurrentDirAndSubDirs()

		// Send those file names to the model for language detection
		prompt := fmt.Sprintf("Use the following files to tell me what languages are being used in this project. Return a comma-separated list with just the language names: %v", files)
		response, err := chatGPTRequest(prompt)

		if err != nil {
			return m, tea.Quit
		}
		m.detectedLanguages = response
		m.isLoadingResponse = false

		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	// Ghost is loading the response from GPT
	if m.isLoadingResponse {
		return indent.String(m.spinner.View()+"Detecting languages...", 2)
	}

	// Ghost has detected languages in the codebase and is asking for confirmation
	if len(m.detectedLanguages) != 0 {
		var yes, no string

		langs := gptResultStyle.Render(m.detectedLanguages)
		title := fmt.Sprintf("%v Ghost detected the following languages in your codebase: %v. Is this correct (y/n)?\n", emoji.Ghost,langs)

		if m.choice == "yes" {
			yes = selectedStyle.Render("> Yes")
			no = itemStyle.Render("No, I want to Ghost to refine its response")
		} else {
			yes = itemStyle.Render("Yes")
			no = selectedStyle.Render("> No, I want to Ghost to refine its response")
		}

		return indent.String(title+yes+"\n"+no, 2)

	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	return ""
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

func chatGPTRequest(prompt string) (response string, err error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "ChatCompletion error", err
	}

	if len(resp.Choices) == 0 {
		return "No languages detected!", err
	} else {
		return resp.Choices[0].Message.Content, nil
	}
}
