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


type errMsg error

type model struct {
	spinner  spinner.Model
	isLoadingResponse bool
	gptResponse string
	quitting bool
	err      error
}

const (
	hotPink = lipgloss.Color("#ff69b7")
)

var (
	gptResultStyle = lipgloss.NewStyle().Foreground(hotPink)
)
func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	default:
		var cmd tea.Cmd
		m.isLoadingResponse = true
		m.spinner, cmd = m.spinner.Update(msg)

		files := getFilesInCurrentDirAndSubDirs()

		prompt := fmt.Sprintf("Use the following files to tell me what languages are being used in this project. Return a comma-separated list with just the language names: %v", files)	
		response, err := chatGPTRequest(files, prompt, m)

		if err != nil {
			return m, tea.Quit
		}
		m.gptResponse = response
		m.isLoadingResponse = false

		return m, cmd
	}
}

func (m model) View() string {
	if m.isLoadingResponse {
		return indent.String(m.spinner.View() + "Detecting languages...", 2)
	}

	if len(m.gptResponse) != 0 {

		langsDetectedMsg := gptResultStyle.Render(m.gptResponse)
		langMsg := fmt.Sprintf("Ghost detected the following languages in your codebase: %v. Is this correct (y/n)?\n", langsDetectedMsg)
		return indent.String(langMsg, 2)
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	return ""
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the ghost CLI",
	Run: func(cmd *cobra.Command, args []string) {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(hotPink)

		m := model {
			spinner: s,
			isLoadingResponse: false,
			gptResponse: "",
			quitting: false,
			err: nil,
		}
		renderWelcome(m)
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			log.Fatal("Error running program: ", err)
			os.Exit(1)
		}
	},
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

func renderWelcome(m model) {
	welcomeMsg := fmt.Sprintf("Running Ghost over your files...  %v\n",  emoji.Ghost)
	welcome := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Render(welcomeMsg)
	fmt.Println(welcome)
}

func chatGPTRequest(files []string, prompt string, m model) (response string, err error) {
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
		return resp.Choices[0].Message.Content,nil
	}
}
