package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/enescakir/emoji"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

type model struct {
	additionalProjectInfo textinput.Model
	choice                string
	currentView           View
	desiredTasks          textinput.Model
	detectedLanguages     string
	err                   error
	GHAWorkflow           string
	quitting              bool
	spinner               spinner.Model
}

const (
	hotPink = lipgloss.Color("#ff69b7")
)

var (
	gptResultStyle = lipgloss.NewStyle().Foreground(hotPink)
	itemStyle      = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(hotPink)
)

type View int64

const (
	ConfirmLanguages         View = 0
	CorrectLanguages         View = 1
	InputTasks               View = 2
	ConfirmTasks             View = 3
	GenerateGHA              View = 4
	CorrectGHA               View = 5
	LoadingDetectedLanguages View = 6
	LoadingGHA               View = 7
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the ghost CLI",
	Run: func(cmd *cobra.Command, args []string) {
		m := initialModel()
		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			log.Fatal("Error running program: ", err)
			os.Exit(1)
		}

	},
}

// Initialize the Bubble Tea model
func initialModel() model {
	// Initialize the spinner for loading
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(hotPink)

	// Initialize the text input for desired tasks in the GHA
	ti := textinput.New()
	ti.Placeholder = "Enter desired tasks to include in your GHA"
	ti.CharLimit = 300
	ti.Width = 300

	additionalInfo := textinput.New()
	additionalInfo.Placeholder = "Enter any additional information about your project"
	additionalInfo.CharLimit = 300
	additionalInfo.Width = 300

	return model{
		additionalProjectInfo: additionalInfo,
		choice:                "yes",
		currentView:           LoadingDetectedLanguages,
		desiredTasks:          ti,
		detectedLanguages:     "",
		err:                   nil,
		GHAWorkflow:           "",
		spinner:               s,
		quitting:              false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// User quits Ghost
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			m.quitting = true
			return m, tea.Quit
		}
		if msg.String() == "up" && m.choice == "no" {
			m.choice = "yes"
		}
		if msg.String() == "down" && m.choice == "yes" {
			m.choice = "no"
		}
		if msg.String() == "enter" {
			if m.currentView == ConfirmLanguages {
				if m.choice == "yes" {
					m.additionalProjectInfo.Blur()
					m.desiredTasks.Focus()
					m.currentView = InputTasks
				} else {
					m.additionalProjectInfo.Focus()
					m.currentView = CorrectLanguages
				}
			}

			if m.currentView == CorrectLanguages && m.additionalProjectInfo.Value() != "" {
				m.currentView = LoadingDetectedLanguages
			}

			if m.currentView == ConfirmTasks {
				m.currentView = GenerateGHA
			}
		}
	}

	if m.currentView == LoadingDetectedLanguages {
		files := getFilesInCurrentDirAndSubDirs()

		var prompt string
		if m.additionalProjectInfo.Value() == "" {
			prompt = fmt.Sprintf("Use the following files to tell me what languages are being used in this project. Return a comma-separated list with just the language names: %v. ", files)
		} else {
			prompt = fmt.Sprintf(`You said this project uses the following languages: %v. 
		According to the user, this is not correct. Here's some additional info from the user: %v.
		Return a comma-separated list of the languages used by this project.`, m.detectedLanguages, m.additionalProjectInfo.Value())
		}
		response, err := chatGPTRequest(prompt)

		if err != nil {
			log.Error(err)
		}
		m.detectedLanguages = response
		m.currentView = ConfirmLanguages
	}

	if m.currentView == ConfirmTasks {
		prompt := fmt.Sprintf(`For a %v program, generate a GitHub Actions workflow that will include the following tasks: %v.
		Leave placeholders for things like version and at the end of generating the GitHub Action, tell the user what their next steps should be`,
			m.detectedLanguages, m.desiredTasks.Value())
		chatGPTStreamingRequest(prompt, m)
		m.currentView = GenerateGHA
	}

	var spinCmd tea.Cmd
	var tasksCmd tea.Cmd
	var additionalInfoCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msg)
	m.desiredTasks, tasksCmd = m.desiredTasks.Update(msg)
	m.additionalProjectInfo, additionalInfoCmd = m.additionalProjectInfo.Update(msg)

	return m, tea.Batch(spinCmd, tasksCmd, additionalInfoCmd)
}

func (m model) View() string {
	if m.currentView == LoadingDetectedLanguages {
		return m.spinner.View() + "Detecting languages..."
	}

	if m.currentView == LoadingGHA {
		return m.spinner.View() + "Generating GHA workflow..."
	}

	if m.currentView == ConfirmLanguages {
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
		}
		return languageConfirmationView(m)
	}

	if m.currentView == CorrectLanguages {
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
		}
		return correctLanguagesView(m)
	}

	if m.currentView == InputTasks {
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
		}
		return giveTasksView(m)
	}

	if m.currentView == GenerateGHA {
		if len(m.detectedLanguages) == 0 || len(m.desiredTasks.Value()) == 0 {
			log.Error("Error: detected languages or desired tasks is empty")
		}
		return fmt.Sprintf("%v Ghost has generated a GitHub Actions workflow for your project: \n\n%v\n\n", emoji.Ghost, m.GHAWorkflow)
	}

	if m.err != nil {
		log.Error("Error: %v\n", m.err)
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

func chatGPTStreamingRequest(prompt string, m model) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Stream: true,
	}
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Error("ChatCompletionStream error: %v\n", err)
		return
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nEnd of response")
			return
		}

		if err != nil {
			log.Error("\nStream error: %v\n", err)
			return
		}

		m.GHAWorkflow = m.GHAWorkflow + response.Choices[0].Delta.Content
		fmt.Printf("%v", response.Choices[0].Delta.Content)
	}
}

func languageConfirmationView(m model) string {
	var yes, no string
	langs := gptResultStyle.Render(m.detectedLanguages)
	title := fmt.Sprintf("%v Ghost detected the following languages in your codebase: %v. Is this correct?\n", emoji.Ghost, langs)

	if m.choice == "yes" {
		yes = selectedStyle.Render("> Yes")
		no = itemStyle.Render("No, I want to Ghost to refine its response")
	} else {
		yes = itemStyle.Render("Yes")
		no = selectedStyle.Render("> No, I want to Ghost to refine its response")
	}

	return title+yes+"\n"+no
}

func giveTasksView(m model) string {
	title := fmt.Sprintf("%v Ghost wants to know if there any specific tasks do you want to do in your GHA (e.g. linting, run tests)?\n", emoji.Ghost)
	return fmt.Sprintf(
		title+"\n%s\n\n%s",
		m.desiredTasks.View(),
		"(Press Enter to continue)",
	) + "\n"
}

func correctLanguagesView(m model) string {
	title := fmt.Sprintf("%v Oops, tell Ghost more about the languages used in your project!\n", emoji.Ghost)
	return fmt.Sprintf(
		title+"\n%s\n\n%s",
		m.additionalProjectInfo.View(),
		"(Press Enter to continue)",
	) + "\n"
}
