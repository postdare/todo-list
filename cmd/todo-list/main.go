// Package main is the entry point for the todo-list TUI application.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"todo-list/internal/storage"
	"todo-list/internal/tui"
)

func main() {
	store, err := storage.NewXDGStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "todo-list: %v\n", err)
		os.Exit(1)
	}
	todos, err := store.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "todo-list: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewModel(store, todos), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "todo-list: %v\n", err)
		os.Exit(1)
	}
}
