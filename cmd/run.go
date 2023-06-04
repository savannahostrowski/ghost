package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	detectedLanguages string
	acceptedLanguages bool
	choice            string
	desiredTasks      textinput.Model
	enteredTasks      bool
	GHAWorkflow       string
	quitting          bool
	err               error
}

const (
	hotPink = lipgloss.Color("#ff69b7")
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
	ti.Focus()
	ti.CharLimit = 300
	ti.Width = 300

	return model{
		spinner:           s,
		isLoadingResponse: true,
		choice:            "yes",
		detectedLanguages: "",
		desiredTasks:      ti,
		GHAWorkflow:       "",
		acceptedLanguages: false,
		enteredTasks:      false,
		quitting:          false,
		err:               nil,
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
			// User accepts or rejects the detected languages
			if len(m.detectedLanguages) != 0 && !m.enteredTasks {
				if m.choice == "yes" {
					m.acceptedLanguages = true
				} else {
					m.acceptedLanguages = false
				}
			}

			if m.acceptedLanguages && len(m.desiredTasks.Value()) != 0  {
				m.enteredTasks = true
				m.isLoadingResponse = true
			}
		}
	}

	if len(m.detectedLanguages) == 0 && !m.acceptedLanguages{
		// Get files in current directory and subdirectories for sending to the model
		files := getFilesInCurrentDirAndSubDirs()

		// Send those file names to the model for language detection
		prompt := fmt.Sprintf("Use the following files to tell me what languages are being used in this project. Return a comma-separated list with just the language names: %v", files)
		response, err := chatGPTRequest(prompt)

		if err != nil {
			log.Error(err)
		}
		m.detectedLanguages = response
		m.isLoadingResponse = false
	}

	var spinCmd tea.Cmd
	var tasksCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msg)
	m.desiredTasks, tasksCmd = m.desiredTasks.Update(msg)

	return m, tea.Batch(spinCmd, tasksCmd)
}

func (m model) View() string {
	// Ghost is loading the response from GPT
	if m.isLoadingResponse {
		// Show spinner on initial language detection
		if len(m.detectedLanguages) == 0 {
			return m.spinner.View() + "Detecting languages..."
		} else {
			// Show spinner on GHA workflow generation
			if m.acceptedLanguages && m.enteredTasks && len(m.GHAWorkflow) == 0 {
				return m.spinner.View() + "Generating GHA workflow..."
			}
		}
	}

	// Ghost has detected languages in the codebase and is asking for confirmation
	if len(m.detectedLanguages) != 0 && !m.acceptedLanguages {
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

		return indent.String(title+yes+"\n"+no, 2)
	}

	// User has accepted the detected languages and is now being asked for desired tasks for the GHA
	if m.acceptedLanguages && !m.enteredTasks {
		title := fmt.Sprintf("%v Ghost wants to know if there any specific tasks do you want to do in your GHA (e.g. linting, run tests)?\n", emoji.Ghost)
		return fmt.Sprintf(
			title+"\n%s\n\n%s",
			m.desiredTasks.View(),
			"(Press Enter to continue)",
		) + "\n"
	}

	// User has accepted the detected languages and has entered desired tasks for the GHA
	if m.enteredTasks && m.acceptedLanguages && len(m.desiredTasks.Value()) != 0 && len(m.detectedLanguages) != 0 {
		prompt := fmt.Sprintf(`For a %v program, generate a GitHub Actions workflow that will include the following tasks: %v.
		Leave placeholders for things like version and at the end of generating the GitHub Action, tell the user what their next steps should be`,
			m.detectedLanguages, m.desiredTasks.Value())
		response, err := chatGPTRequest(prompt)
		m.GHAWorkflow = response

		if err != nil {
			log.Error("Error: %v\n", err)
		}

		return fmt.Sprintf("Here is your GitHub Actions workflow:\n\n%v\n\n", m.GHAWorkflow)

	}

	if m.err != nil {
		log.Error("Error: %v\n", m.err)
	}
	return fmt.Sprintf("%v", m.enteredTasks)
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
