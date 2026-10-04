package main

import (
	"fmt"
	"os"

	"github.com/M5Devs/Total-Connect/internal/config"
	"github.com/M5Devs/Total-Connect/internal/core"
	"github.com/M5Devs/Total-Connect/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfgPath := config.GetConfigPath()
	engine, err := core.NewRcloneEngine(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage engine: %v\n", err)
		os.Exit(1)
	}

	appModel := tui.NewAppModel(engine)
	p := tea.NewProgram(appModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI application: %v\n", err)
		os.Exit(1)
	}
}
