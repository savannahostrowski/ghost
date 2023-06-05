package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

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
	files                 []string
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
	Goodbye                  View = 8
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

			if m.currentView == InputTasks && m.desiredTasks.Value() != "" {
				m.desiredTasks.Blur()
				m.currentView = LoadingGHA
			}

			if m.currentView == GenerateGHA {
				if m.choice == "yes" {
					writeGHAWorkflowToFile(m.GHAWorkflow)
					m.currentView = Goodbye
				} else {
					m.additionalProjectInfo.Focus()
					m.currentView = CorrectGHA
				}
			}
		}
	}

	if m.currentView == Goodbye {
		return m, tea.Quit
	}

	if m.currentView == LoadingDetectedLanguages {
		var prompt string
		if m.additionalProjectInfo.Value() == "" {
			files := getFilesInCurrentDirAndSubDirs()
			m.files = files
			prompt = fmt.Sprintf("Use the following files to tell me what languages are being used in this project. Return a comma-separated list with just the language names: %v. ", files)
		} else {
			prompt = fmt.Sprintf(`You said this project uses the following languages %v (detected from the following files: %v). 
		According to the user, this is not correct. Here's some additional info from the user: %v.
		Return a comma-separated list of the languages used by this project.`, m.files, m.detectedLanguages, m.additionalProjectInfo.Value())
		}
		response, err := chatGPTRequest(prompt)

		if err != nil {
			log.Error(err)
		}
		m.detectedLanguages = response
		m.additionalProjectInfo.SetValue("")
		m.currentView = ConfirmLanguages
	}

	if m.currentView == LoadingGHA {
		prompt := fmt.Sprintf(`For a %v program, generate a GitHub Actions workflow that will include the following tasks: %v.
		Name it "Ghost-generated pipeline". Have it run on push to master or main, unless the user specified otherwise.
		Leave placeholders for things like version and at the end of generating the GitHub Action, tell the user what their next steps should be`,
			m.detectedLanguages, m.desiredTasks.Value())
		response, err := chatGPTRequest(prompt)
		if err != nil {
			log.Error(err)
		}
		m.GHAWorkflow = response
		m.desiredTasks.SetValue("")
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
		return m.spinner.View() + "Generating a GitHub Actions workflow...This might take a couple minutes."
	}

	if m.currentView == ConfirmLanguages {
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
			return ""
		}
		return languageConfirmationView(m)
	}

	if m.currentView == CorrectLanguages {
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
			return ""
		}
		return correctLanguagesView(m)
	}

	if m.currentView == InputTasks {
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
			return ""
		}
		return giveTasksView(m)
	}

	if m.currentView == GenerateGHA {
		if len(m.GHAWorkflow) == 0 {
			log.Error("Error: detected languages or desired tasks is empty")
			return ""
		}
		return generateGHAView(m)
	}

	if m.currentView == CorrectGHA {
		if len(m.GHAWorkflow) == 0 {
			log.Error("Error: detected languages or desired tasks is empty")
			return ""
		}
		return correctGHAView(m)
	}

	if m.err != nil {
		log.Error("Error: %v\n", m.err)
	}
	return fmt.Sprintf("%v You successfully generated a GitHub Action workflow with Ghost! Goodbye!", emoji.Ghost)
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

	return title + yes + "\n" + no
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

func generateGHAView(m model) string {
	var yes, no string
	title := fmt.Sprintf("%v Ghost generated this GitHub Actions workflow for your project...\n\n\n", emoji.Ghost)

	if m.choice == "yes" {
		yes = selectedStyle.Render("> Great! Output to .github/workflows/ghost.yml")
		no = itemStyle.Render("I want Ghost to refine to generated GHA workflow")
	} else {
		yes = itemStyle.Render("Great! Output to .github/workflows/ghost.yml")
		no = selectedStyle.Render("> I want Ghost to refine to generated GHA workflow")
	}

	return title +
	"----------------------------------------\n" +
	 gptResultStyle.Render(m.GHAWorkflow) + "\n" +
	 "----------------------------------------\n" +
	 "How does this look?" + "\n" + yes + "\n" + no
}

func correctGHAView(m model) string {
	title := fmt.Sprintf("%v Oops, tell Ghost more about the tasks you want to do in your GHA!\n", emoji.Ghost)
	return fmt.Sprintf(
		title+"\n%s\n\n%s",
		m.additionalProjectInfo.View(),
		"(Press Enter to continue)",
	) + "\n"
}

func writeGHAWorkflowToFile(gha string) {
	_, err := os.Stat(".github/workflows")
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(".github/workflows", 0755)
		if errDir != nil {
			log.Error("Error creating .github/workflows directory")
			return
		}
	}

	filename := fmt.Sprintf(".github/workflows/ghost_%v.yml", time.Now().UnixNano())
	_, err = os.Create(filename)
	if err != nil {
		log.Error("Error creating ghost.yml file")
		return
	}
	
	ioutil.WriteFile(filename, []byte(gha), 0644)
}