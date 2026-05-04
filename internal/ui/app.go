package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhermoso/ghtui/internal/config"
	gh "github.com/jhermoso/ghtui/internal/github"
	"github.com/jhermoso/ghtui/internal/ui/components"
	"github.com/jhermoso/ghtui/internal/ui/keymap"
	"github.com/jhermoso/ghtui/internal/ui/views"
)

type viewID int

const (
	viewAuth viewID = iota
	viewDashboard
)

type App struct {
	activeView viewID
	width      int
	height     int

	authView  views.AuthView
	dashboard views.Dashboard

	client *gh.Client
	user   string

	helpView components.HelpOverlay
	cfg      *config.Config
}

func NewApp(cfg *config.Config, token string) *App {
	a := &App{cfg: cfg}

	if token == "" {
		a.activeView = viewAuth
		a.authView = views.NewAuthView(cfg)
	} else {
		a.client = gh.NewClient(token)
		a.activeView = viewDashboard
		a.dashboard = views.NewDashboard(a.client, "")
	}

	a.helpView = components.NewHelpOverlay(keymap.List)
	return a
}

func (a *App) SetUser(login string) {
	a.user = login
	if a.activeView == viewDashboard {
		a.dashboard = views.NewDashboard(a.client, login)
	}
}

func (a App) Init() tea.Cmd {
	switch a.activeView {
	case viewAuth:
		return a.authView.Init()
	case viewDashboard:
		return a.dashboard.Init()
	}
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.helpView.SetWidth(msg.Width)

	case views.AuthSuccessMsg:
		a.client = gh.NewClient(msg.Token)
		a.user = msg.Login
		a.activeView = viewDashboard
		a.dashboard = views.NewDashboard(a.client, msg.Login)
		return a, a.dashboard.Init()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "?":
			a.helpView.Toggle()
			return a, nil
		}
	}

	if a.helpView.Visible() {
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "?" {
			a.helpView.Toggle()
			return a, nil
		}
		return a, nil
	}

	var cmd tea.Cmd
	switch a.activeView {
	case viewAuth:
		var updated views.AuthView
		updated, cmd = a.authView.Update(msg)
		a.authView = updated
	case viewDashboard:
		var updated views.Dashboard
		updated, cmd = a.dashboard.Update(msg)
		a.dashboard = updated
	}
	return a, cmd
}

func (a App) View() string {
	if a.helpView.Visible() {
		overlay := a.helpView.View()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	switch a.activeView {
	case viewAuth:
		return a.authView.View()
	case viewDashboard:
		return a.dashboard.View()
	}
	return ""
}
