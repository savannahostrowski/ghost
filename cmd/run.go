package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/enescakir/emoji"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	viewport              viewport.Model
}

const (
	hotPink = lipgloss.Color("#ff69b7")
	purple  = lipgloss.Color("#bd93f9")
	red     = lipgloss.Color("#ff5555")
	grey    = lipgloss.Color("#44475a")
)

var (
	gptResultStyle = lipgloss.NewStyle().Foreground(hotPink)
	userInputStyle = lipgloss.NewStyle().Foreground(purple)
	itemStyle      = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(purple)
	errorStyle     = lipgloss.NewStyle().Foreground(red)
	helpStyle      = lipgloss.NewStyle().Foreground(grey)
	viewportStyle  = lipgloss.NewStyle().Foreground(hotPink)
)

type View int64

const (
	ConfirmLanguages View = iota
	CorrectLanguages
	InputTasks
	ConfirmTasks
	GenerateGHA
	CorrectGHA
	LoadingDetectedLanguages
	LoadingGHA
	Goodbye
	Preload
	Error
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Ghost CLI",
	Run: func(cmd *cobra.Command, args []string) {
		err := viper.ReadInConfig()
		if err != nil {
			panic(fmt.Errorf("fatal error config file: %w", err))
		}

		if !viper.IsSet("OPENAI_API_KEY") {
			log.Error("Please set OPENAI_API_KEY in your environment using `ghost config set OPENAI_API_KEY <your key>`")
			os.Exit(1)
		}

		m := initialModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
		if _, err := p.Run(); err != nil {
			log.Fatal("Yikes! We've run into a problem: ", err)
			os.Exit(1)
		}
	},
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(hotPink)

	ti := textinput.New()
	ti.Placeholder = "Enter some tasks"
	ti.CharLimit = 300
	ti.Width = 300

	additionalInfo := textinput.New()
	additionalInfo.Placeholder = "Tell Ghost more about your project"
	additionalInfo.CharLimit = 300
	additionalInfo.Width = 300

	vp := viewport.New(0, 0)
	vp.Style = viewportStyle
	vp.YPosition = 0
	vp.KeyMap.Up = key.NewBinding(key.WithKeys("w"))
	vp.KeyMap.Down = key.NewBinding(key.WithKeys("s"))

	return model{
		additionalProjectInfo: additionalInfo,
		choice:                "yes",
		currentView:           Preload,
		desiredTasks:          ti,
		detectedLanguages:     "",
		err:                   nil,
		GHAWorkflow:           "",
		spinner:               s,
		quitting:              false,
		viewport:              vp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case gptResponse:
		switch m.currentView {
		case LoadingDetectedLanguages:
			m.detectedLanguages = string(msg)
			m.additionalProjectInfo.SetValue("")
			m.currentView = ConfirmLanguages
		case LoadingGHA:
			m.GHAWorkflow = string(msg)
			m.viewport.SetContent(m.GHAWorkflow)
			m.currentView = GenerateGHA
		default:
			panic(fmt.Sprintf("unexpected view: %v", m.currentView))
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up":
			if m.choice == "no" {
				m.choice = "yes"
			}
		case "down":
			if m.choice == "yes" {
				m.choice = "no"
			}
		case "enter":
			switch m.currentView {
			// Confirm that the detected languages are correct
			case ConfirmLanguages:
				if m.choice == "yes" {
					m.additionalProjectInfo.Blur()
					m.desiredTasks.Focus()
					m.currentView = InputTasks
				} else {
					m.additionalProjectInfo.Focus()
					m.currentView = CorrectLanguages
				}
			// Correct the detected languages (when Ghost was wrong or more details are needed)
			case CorrectLanguages:
				if m.additionalProjectInfo.Value() != "" {
					m.currentView = Preload
					m.choice = "yes"
				}
			// Allow the user to specify what tasks they'd like to do in their pipeline
			case InputTasks, CorrectGHA:
				if m.desiredTasks.Value() != "" {
					m.currentView = LoadingGHA
					cmds = append(cmds, func() tea.Msg {
						prompt := fmt.Sprintf(`For a %v program, generate a GitHub Action workflow that will include the following tasks: %v.
						Name it "Ghost-generated pipeline". Leave placeholders for things like version and at the end of generating the
						GitHub Action, tell the user what their next steps should be in a comment`, m.detectedLanguages, m.desiredTasks.Value())
						response, err := chatGPTRequest(prompt)
						if err != nil {
							// log.Error(err)
							os.Exit(1)
						}
						return response
					})
				}

			case GenerateGHA:
				if m.choice == "yes" {
					writeGHAWorkflowToFile(m.GHAWorkflow)
					m.currentView = Goodbye
				} else {
					m.additionalProjectInfo.Focus()
					m.currentView = CorrectGHA
				}
			}
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - msg.Height/3
	default:
		switch m.currentView {
		case Preload:

			if len(m.files) == 0 {
				m.files = getFilesInCurrentDirAndSubDirs()
			}

			m.currentView = LoadingDetectedLanguages
			cmds = append(cmds, func() tea.Msg {
				var prompt string
				if m.additionalProjectInfo.Value() == "" {
					prompt = fmt.Sprintf("Use the following files to tell me what languages are being used in this project. Return a comma-separated list with just the language names: %v. ", m.files)
				} else {
					prompt = fmt.Sprintf(`You said this project uses the following languages %v (detected from the following files: %v). 
			According to the user, this is not correct. Here's some additional info from the user: %v.
			Return a comma-separated list of the languages used by this project.`, m.files, m.detectedLanguages, m.additionalProjectInfo.Value())
				}
				response, err := chatGPTRequest(prompt)

				if err != nil {
					log.Error(err)
					os.Exit(1)
				}
				return response
			})
		case Goodbye:
			time.Sleep(1 * time.Second)
			return m, tea.Quit
		}
	}
	var spinCmd tea.Cmd
	var tasksCmd tea.Cmd
	var additionalInfoCmd tea.Cmd
	var viewportCmd tea.Cmd
	m.viewport, viewportCmd = m.viewport.Update(msg)
	m.spinner, spinCmd = m.spinner.Update(msg)
	m.desiredTasks, tasksCmd = m.desiredTasks.Update(msg)
	m.additionalProjectInfo, additionalInfoCmd = m.additionalProjectInfo.Update(msg)

	cmds = append(cmds, spinCmd, tasksCmd, additionalInfoCmd, viewportCmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.currentView {
	case LoadingDetectedLanguages:
		return m.spinner.View() + "Detecting languages..."
	case LoadingGHA:
		return m.spinner.View() + "Generating a GitHub Action workflow...This might take a couple of minutes."
	case ConfirmLanguages:
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
			return ""
		}
		return confirmationView(
			m,
			fmt.Sprintf("%v Ghost detected the following languages in your codebase: %v. Is this correct?\n", emoji.Ghost, gptResultStyle.Render(m.detectedLanguages)),
			"Yes",
			"No - I want to correct the language(s) Ghost detected",
			false,
			"")
	case CorrectLanguages:
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
			return ""
		}
		return textInputView(m, fmt.Sprintf("%v Oops! Let's try again. What languages are being used in this project?", emoji.Ghost), m.additionalProjectInfo)
	case InputTasks:
		if len(m.detectedLanguages) == 0 {
			log.Error("Error: detected languages is empty")
			return ""
		}
		return textInputView(m, fmt.Sprintf("%v What tasks should Ghost include in your GitHub Action workflow?", emoji.Ghost), m.desiredTasks)
	case GenerateGHA:
		if len(m.GHAWorkflow) == 0 {
			log.Error("Error: detected languages or desired tasks is empty")
			return ""
		}
		return confirmationView(m,
			"",
			"Great! Output to .github/workflows/ghost.yml",
			"I want Ghost to refine to generated GitHub Action workflow",
			true,
			m.GHAWorkflow)
	case CorrectGHA:
		if len(m.GHAWorkflow) == 0 {
			log.Error("Error: detected languages or desired tasks is empty")
			return ""
		}
		return textInputView(m, "Oops! Let's try again. Update the tasks should be included in the GitHub Action workflow?", m.desiredTasks)
	case Goodbye:
		return fmt.Sprintf("%v You successfully generated a GitHub Action workflow with Ghost (in .github/workflows/). Goodbye!", emoji.Ghost)
	case Error:
		return errorView(m)
	default:
		return ""
	}
}

type gptResponse string

func chatGPTRequest(prompt string) (response gptResponse, err error) {
	viper.ReadInConfig()
	apiKey := viper.GetString("openai_api_key")
	enableGPT4, _ := strconv.ParseBool(viper.GetString("enable_gpt_4"))
	var model string
	if enableGPT4 {
		model = openai.GPT4
	} else {
		model = openai.GPT3Dot5Turbo
	}
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
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
		return gptResponse(resp.Choices[0].Message.Content), nil
	}
}

func textInputView(m model, title string, input textinput.Model) string {
	return fmt.Sprintf(
		title+"\n\n%s\n\n%s",
		userInputStyle.Render(input.View()),
		"(Press "+userInputStyle.Render("Enter")+" to continue)",
	) + "\n"
}

func confirmationView(m model, title string, yesText string, noText string, isGHAOutput bool, content string) string {
	var yes, no string
	if m.choice == "yes" {
		yes = selectedStyle.Render("> " + yesText)
		no = itemStyle.Render(noText)
	} else {
		yes = itemStyle.Render(yesText)
		no = selectedStyle.Render("> " + noText)
	}

	if isGHAOutput {
		return title +
			m.viewport.View() + "\n" +
			helpStyle.Render("   ↑w/↓s: Scroll (or use your mouse) \n") + "\n" +
			fmt.Sprintf("%v Ghost created this GitHub Action. How does it look?", emoji.Ghost) + "\n" + yes + "\n" + no
	} else {
		return title + yes + "\n" + no
	}
}

func errorView(m model) string {
	return errorStyle.Render(m.err.Error())
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

	os.WriteFile(filename, []byte(gha), 0644)
}
