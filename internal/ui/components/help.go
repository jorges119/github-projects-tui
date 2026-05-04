package components

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

var helpStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#89b4fa")).
	Padding(1, 2)

type HelpOverlay struct {
	help    help.Model
	keys    help.KeyMap
	visible bool
	width   int
}

func NewHelpOverlay(keys help.KeyMap) HelpOverlay {
	h := help.New()
	h.ShowAll = true
	return HelpOverlay{help: h, keys: keys}
}

func (h *HelpOverlay) Toggle() { h.visible = !h.visible }
func (h HelpOverlay) Visible() bool { return h.visible }
func (h *HelpOverlay) SetWidth(w int) { h.width = w; h.help.Width = w - 6 }

func (h HelpOverlay) View() string {
	if !h.visible {
		return ""
	}
	content := h.help.FullHelpView(h.keys.FullHelp())
	return helpStyle.Render(content)
}
