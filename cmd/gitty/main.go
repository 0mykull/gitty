package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/0mykull/gitty/internal/config"
	"github.com/0mykull/gitty/internal/styles"
	"github.com/0mykull/gitty/internal/ui"
)

func main() {
	// Load or create config
	cfg, err := config.EnsureConfig()
	if err != nil {
		fmt.Printf("%s Failed to load config: %v\n", styles.Icons.Cross, err)
		os.Exit(1)
	}

	// Create and run the program
	model := ui.NewModel(cfg)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Printf("%s Error: %v\n", styles.Icons.Cross, err)
		os.Exit(1)
	}
}
