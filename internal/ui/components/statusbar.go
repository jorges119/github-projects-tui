package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e2e")).
			Foreground(lipgloss.Color("#cdd6f4"))

	statusLeftStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#89b4fa")).
			Foreground(lipgloss.Color("#1e1e2e")).
			Bold(true).
			Padding(0, 1)

	statusRightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#313244")).
			Foreground(lipgloss.Color("#a6adc8")).
			Padding(0, 1)

	statusMidStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e2e")).
			Foreground(lipgloss.Color("#cdd6f4")).
			Padding(0, 1)
)

type StatusBar struct {
	Width   int
	User    string
	View    string
	Context string
	Hint    string
}

func (s StatusBar) Render() string {
	left := statusLeftStyle.Render(fmt.Sprintf(" %s", s.View))
	right := statusRightStyle.Render(s.User)

	hintText := s.Hint
	if hintText == "" {
		hintText = "? help"
	}
	hint := statusRightStyle.Render(hintText)

	mid := s.Context
	if mid == "" {
		mid = ""
	}

	midRendered := statusMidStyle.Render(mid)

	usedWidth := lipgloss.Width(left) + lipgloss.Width(right) + lipgloss.Width(hint) + lipgloss.Width(midRendered)
	padding := s.Width - usedWidth
	if padding < 0 {
		padding = 0
	}

	spacer := statusBarStyle.Width(padding).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Bottom,
		left, midRendered, spacer, hint, right,
	)
}
