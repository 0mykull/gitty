package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/0mykull/gitty/internal/config"
	"github.com/0mykull/gitty/internal/git"
	"github.com/0mykull/gitty/internal/styles"
	"github.com/0mykull/gitty/internal/ui"
)

func main() {
	// Check dependencies
	missing := git.CheckDeps()
	for _, m := range missing {
		if strings.Contains(m, "(required)") {
			fmt.Printf("%s Missing dependency: %s\n", styles.Icons.Cross, m)
			os.Exit(1)
		}
		// Just warn for optional
		// fmt.Printf("%s Warning: %s not found\n", styles.Icons.Warning, m)
	}

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
