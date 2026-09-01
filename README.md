# todo-list

A minimal but polished Go CLI TUI for managing a personal todo list, built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

Todos are stored as human-readable JSON in the XDG config directory
(`~/.config/todo-list/todos.json` by default), so they survive restarts.

## Features

- Three status tabs: **未开始** / **进行中** / **已完成**
- Add todos with `n` (Enter saves, Esc cancels)
- Toggle completion with `Space` (已完成 ↔ 未开始)
- Mark in-progress with `i`
- ↑/↓ navigate the current tab, ←/→ switch tabs
- `q` or `Ctrl+C` quits cleanly
- Lip Gloss styling with a persistent help footer
- Storage is behind an interface, so future providers can be swapped in

## Keybindings

| Key          | Action                              |
| ------------ | ----------------------------------- |
| ↑ / ↓        | Move selection within the tab       |
| ← / →        | Switch tab                          |
| `n`          | Add a new todo (Enter save, Esc cancel) |
| `Space`      | Toggle 已完成 / 未开始              |
| `i`          | Mark the selected todo 进行中       |
| `q` / Ctrl+C | Quit                                |

## Build & run

```sh
go build -o bin/todo-list ./cmd/todo-list
./bin/todo-list
```

## Smoke commands

```sh
go build ./...                      # must succeed
go test ./...                       # unit + PTY end-to-end tests
./bin/todo-list                     # launches the TUI
cat ~/.config/todo-list/todos.json  # human-readable JSON after adding/modifying todos
```

## Project layout

```
cmd/todo-list/    entry point (wires storage + TUI)
internal/todo/    domain model (Todo, Status)
internal/storage/ Storage interface + JSON file implementation
internal/tui/     Bubble Tea model, views, key handling
```
