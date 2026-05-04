package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhermoso/ghtui/internal/auth"
	"github.com/jhermoso/ghtui/internal/config"
	"github.com/jhermoso/ghtui/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	token, err := auth.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading token: %v\n", err)
		os.Exit(1)
	}

	// Allow token override via environment variable
	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" && token == "" {
		token = envToken
	}

	app := ui.NewApp(cfg, token)

	if token != "" {
		user, err := auth.ValidateToken(token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stored token is invalid: %v\nRun ghtui again to re-authenticate.\n", err)
			if delErr := auth.DeleteToken(); delErr != nil {
				fmt.Fprintf(os.Stderr, "error deleting token: %v\n", delErr)
			}
			os.Exit(1)
		}
		app.SetUser(user.Login)
	}

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running program: %v\n", err)
		os.Exit(1)
	}
}
