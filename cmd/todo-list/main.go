// Package main is the entry point for the todo-list application. By default
// it launches the terminal UI; with -web it serves the same list over HTTP.
package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"todo-list/internal/storage"
	"todo-list/internal/tui"
	"todo-list/internal/web"
)

func main() {
	webMode := flag.Bool("web", false, "启动 Web UI（浏览器访问）而不是终端 TUI")
	addr := flag.String("addr", "127.0.0.1:8080", "Web 模式下的监听地址")
	flag.Parse()

	store, err := storage.NewXDGStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "todo-list: %v\n", err)
		os.Exit(1)
	}

	if *webMode {
		runWeb(store, *addr)
		return
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

// runWeb serves the web UI until the process is terminated.
func runWeb(store storage.Storage, addr string) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           web.NewServer(store),
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("todo-list: Web UI 已启动：http://%s\n", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "todo-list: %v\n", err)
		os.Exit(1)
	}
}
