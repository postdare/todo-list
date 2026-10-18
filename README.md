# todo-list

A minimal but polished Go todo list with two front ends: a terminal UI (TUI)
built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss), and an optional web UI
served over HTTP. Both share the same storage.

Todos are stored as human-readable JSON in the XDG config directory
(`~/.config/todo-list/todos.json` by default), so they survive restarts.
User-defined web panels are stored next to it in `panels.json`; each todo's
`panel_id` lives in `todos.json`. Legacy `todos.json` files (a plain array
without `panel_id`) load unchanged and are treated as ungrouped.

## Features

- Three status tabs: **未开始** / **进行中** / **已完成**
- Web UI: create, rename, and delete custom panels (自定义分区); assign a todo
  to a panel with the dropdown on its card. Deleting a panel moves its todos
  back to the status columns.
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
./bin/todo-list          # terminal UI
./bin/todo-list -web     # web UI at http://127.0.0.1:8080
```

The web UI listens on `127.0.0.1:8080` by default; change it with
`-addr <host:port>`:

```sh
./bin/todo-list -web -addr 0.0.0.0:9000
```

Web mode serves a browser page plus a JSON API over the same
`todos.json` used by the TUI:

| Method & path            | Purpose                                      |
| ------------------------ | -------------------------------------------- |
| `GET /`                  | Browser UI                                   |
| `GET /api/todos`         | List all todos                               |
| `POST /api/todos`        | Create `{title, description}`                |
| `PUT /api/todos/{id}`    | Partially update `{title, description, status, panel_id}`; empty `panel_id` ungroups |
| `DELETE /api/todos/{id}` | Delete a todo                                |
| `GET /api/panels`        | List custom panels                           |
| `POST /api/panels`       | Create a panel from `{name}`                 |
| `PUT /api/panels/{id}`   | Rename a panel from `{name}`                 |
| `DELETE /api/panels/{id}`| Delete a panel and ungroup its todos         |

Errors use the same shape across the API: `{"error": "..."}`.

## Smoke commands

```sh
go build ./...                      # must succeed
go test ./...                       # unit + PTY end-to-end tests
./bin/todo-list                     # launches the TUI
cat ~/.config/todo-list/todos.json  # human-readable JSON after adding/modifying todos
cat ~/.config/todo-list/panels.json # custom panels (absent until one is created)
```

## Project layout

```
cmd/todo-list/    entry point (wires storage + TUI / web server)
internal/todo/    domain model (Todo, Panel, Status)
internal/storage/ Storage interface + JSON file implementation (todos.json, panels.json)
internal/tui/     Bubble Tea model, views, key handling
internal/web/     HTTP web UI (embedded HTML) and JSON API
```
