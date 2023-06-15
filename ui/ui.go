package ui

import "github.com/charmbracelet/lipgloss"


const (
	HotPink = lipgloss.Color("#ff69b7")
	Purple  = lipgloss.Color("#bd93f9")
	Red     = lipgloss.Color("#ff5555")
	Grey    = lipgloss.Color("#44475a")
)

var (
	GptResultStyle = lipgloss.NewStyle().Foreground(HotPink)
	UserInputStyle = lipgloss.NewStyle().Foreground(Purple)
	ItemStyle      = lipgloss.NewStyle().PaddingLeft(2)
	SelectedStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(Purple)
	ErrorStyle     = lipgloss.NewStyle().Foreground(Red)
	HelpStyle      = lipgloss.NewStyle().Foreground(Grey)
	ViewportStyle  = lipgloss.NewStyle().Foreground(HotPink)
)