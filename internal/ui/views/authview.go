package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhermoso/ghtui/internal/auth"
	"github.com/jhermoso/ghtui/internal/config"
)

type AuthMode int

const (
	AuthModeSelect AuthMode = iota
	AuthModePAT
	AuthModeDevice
)

type AuthSuccessMsg struct {
	Token string
	Login string
}

type AuthErrorMsg struct{ Err error }

type deviceCodeReadyMsg struct {
	UserCode        string
	VerificationURI string
	DeviceCode      string
	ClientID        string
}

type tokenReceivedMsg struct{ Token string }

var (
	authTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#89b4fa")).
			MarginBottom(1)

	authOptionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cdd6f4")).
			PaddingLeft(2)

	authSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6e3a1")).
				Bold(true).
				PaddingLeft(2)

	authErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f38ba8"))

	authSubtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086"))

	authBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#313244")).
			Padding(1, 2).
			Width(60)
)

type AuthView struct {
	mode     AuthMode
	cursor   int
	input    textinput.Model
	status   string
	errMsg   string
	width    int
	height   int
	cfg      *config.Config
	waiting  bool
	userCode string
	verifyURI string
}

func NewAuthView(cfg *config.Config) AuthView {
	ti := textinput.New()
	ti.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxx"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 50
	ti.CharLimit = 256

	return AuthView{cfg: cfg, input: ti}
}

func (v AuthView) Init() tea.Cmd { return nil }

func (v AuthView) Update(msg tea.Msg) (AuthView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height

	case tea.KeyMsg:
		if v.waiting {
			return v, nil
		}
		switch v.mode {
		case AuthModeSelect:
			switch msg.String() {
			case "j", "down":
				v.cursor = (v.cursor + 1) % 2
			case "k", "up":
				v.cursor = (v.cursor - 1 + 2) % 2
			case "enter":
				if v.cursor == 0 {
					v.mode = AuthModePAT
					v.input.Focus()
					return v, textinput.Blink
				}
				v.mode = AuthModeDevice
				return v, v.startDeviceFlow()
			}

		case AuthModePAT:
			switch msg.String() {
			case "enter":
				token := strings.TrimSpace(v.input.Value())
				if token == "" {
					v.errMsg = "token cannot be empty"
					return v, nil
				}
				v.waiting = true
				v.status = "Validating token..."
				v.errMsg = ""
				return v, v.validatePAT(token)
			case "esc":
				v.mode = AuthModeSelect
				v.input.SetValue("")
				v.errMsg = ""
				return v, nil
			}

		case AuthModeDevice:
			if msg.String() == "esc" {
				v.mode = AuthModeSelect
				v.errMsg = ""
				v.userCode = ""
				v.verifyURI = ""
				return v, nil
			}
		}

	case deviceCodeReadyMsg:
		v.waiting = false
		v.userCode = msg.UserCode
		v.verifyURI = msg.VerificationURI
		v.status = "Waiting for authorization..."
		return v, v.pollToken(msg.ClientID, msg.DeviceCode)

	case tokenReceivedMsg:
		v.waiting = true
		v.status = "Validating token..."
		return v, v.validatePAT(msg.Token)

	case AuthSuccessMsg:
		v.waiting = false

	case AuthErrorMsg:
		v.waiting = false
		v.errMsg = msg.Err.Error()
		v.status = ""
	}

	if v.mode == AuthModePAT {
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v AuthView) View() string {
	var b strings.Builder

	b.WriteString(authTitleStyle.Render("  GitHub Authentication") + "\n\n")

	switch v.mode {
	case AuthModeSelect:
		b.WriteString(authSubtleStyle.Render("Choose an authentication method:") + "\n\n")
		options := []string{"Personal Access Token (PAT)", "Device Flow (OAuth App)"}
		for i, opt := range options {
			if i == v.cursor {
				b.WriteString(authSelectedStyle.Render("▶ " + opt) + "\n")
			} else {
				b.WriteString(authOptionStyle.Render("  " + opt) + "\n")
			}
		}
		b.WriteString("\n" + authSubtleStyle.Render("j/k to move, enter to select") + "\n")

	case AuthModePAT:
		b.WriteString(authSubtleStyle.Render("Paste your Personal Access Token:") + "\n")
		b.WriteString(authSubtleStyle.Render("(Settings → Developer Settings → Personal access tokens)") + "\n\n")
		b.WriteString(authSubtleStyle.Render("Required scopes: repo, project, read:org") + "\n\n")
		b.WriteString(v.input.View() + "\n\n")
		b.WriteString(authSubtleStyle.Render("enter to confirm • esc to go back") + "\n")

	case AuthModeDevice:
		if v.userCode == "" {
			b.WriteString(authSubtleStyle.Render("Starting device authorization flow...") + "\n")
			if v.cfg.ClientID == "" {
				b.WriteString("\n" + authErrorStyle.Render("GHTUI_CLIENT_ID not set.") + "\n")
				b.WriteString(authSubtleStyle.Render("Create a GitHub OAuth App and set the env var:") + "\n")
				b.WriteString(authSubtleStyle.Render("  export GHTUI_CLIENT_ID=<your-client-id>") + "\n")
			}
		} else {
			b.WriteString(fmt.Sprintf("1. Open: %s\n\n", authSelectedStyle.Render(v.verifyURI)))
			b.WriteString(fmt.Sprintf("2. Enter code: %s\n\n",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fab387")).Render(v.userCode)))
			b.WriteString(authSubtleStyle.Render("Waiting for you to authorize in the browser...") + "\n")
			b.WriteString(authSubtleStyle.Render("esc to cancel") + "\n")
		}
	}

	if v.status != "" {
		b.WriteString("\n" + authSubtleStyle.Render(v.status) + "\n")
	}
	if v.errMsg != "" {
		b.WriteString("\n" + authErrorStyle.Render("✗ "+v.errMsg) + "\n")
	}

	box := authBoxStyle.Render(b.String())
	return lipgloss.Place(v.width, v.height, lipgloss.Center, lipgloss.Center, box)
}

func (v AuthView) startDeviceFlow() tea.Cmd {
	return func() tea.Msg {
		if v.cfg.ClientID == "" {
			return AuthErrorMsg{Err: fmt.Errorf("GHTUI_CLIENT_ID not set (see above)")}
		}
		dc, err := auth.RequestDeviceCode(v.cfg.ClientID)
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		return deviceCodeReadyMsg{
			UserCode:        dc.UserCode,
			VerificationURI: dc.VerificationURI,
			DeviceCode:      dc.DeviceCode,
			ClientID:        v.cfg.ClientID,
		}
	}
}

func (v AuthView) pollToken(clientID, deviceCode string) tea.Cmd {
	return func() tea.Msg {
		dc := &auth.DeviceCode{DeviceCode: deviceCode, ExpiresIn: 900, Interval: 5}
		token, err := auth.PollForToken(context.Background(), clientID, dc)
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		return tokenReceivedMsg{Token: token}
	}
}

func (v AuthView) validatePAT(token string) tea.Cmd {
	return func() tea.Msg {
		user, err := auth.ValidateToken(token)
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		if err := auth.SaveToken(token); err != nil {
			return AuthErrorMsg{Err: fmt.Errorf("saving token: %w", err)}
		}
		return AuthSuccessMsg{Token: token, Login: user.Login}
	}
}
