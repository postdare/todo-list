# Validation contract — Go TUI Todo List MVP

Mission: `m-202608311424-fl3zgc`

## Objective

Build a minimal but polished Go CLI TUI for managing a personal todo list. The MVP supports three status tabs (未开始 / 进行中 / 已完成), keyboard navigation, adding todos, toggling completion with Space, marking in-progress with `i`, and local JSON persistence. The storage layer is designed behind an interface so future providers can be swapped in.

## Behavioral assertions

1. `go build ./...` succeeds without errors.
2. Running the binary launches a TUI with three tabs labeled 未开始, 进行中, and 已完成.
3. Pressing `n` opens a prompt to enter a new todo title; pressing Enter saves it and it appears under the 未开始 tab.
4. Pressing ↑/↓ moves the selection indicator through todos in the current tab.
5. Pressing ←/→ switches the active tab and updates the filtered list.
6. Pressing Space on a non-已完成 todo marks it 已完成; pressing Space on a 已完成 todo marks it 未开始.
7. Pressing `i` on a selected todo marks it 进行中.
8. Todos are persisted to a JSON file in the XDG config directory; after quitting and relaunching, the same todos appear in the correct tabs.
9. Pressing `q` or Ctrl+C exits the application cleanly.
10. The JSON storage file is human-readable and contains id, title, status, and created_at for each todo.

## Smoke commands

- `go build -o bin/todo-list ./cmd/todo-list`
- `./bin/todo-list`
- `cat ~/.config/todo-list/todos.json  # after adding/ modifying todos`

## Milestones

### ms1: Foundation & Local Storage

- **f1** (pending) Go module and project layout
- **f2** (pending) Todo domain model and JSON storage interface
- **f3** (pending) Storage unit tests

### ms2: TUI Core Layout & Navigation

- **f4** (pending) Bubble Tea app scaffold
- **f5** (pending) Render tabs and filtered todo list
- **f6** (pending) Keyboard navigation

### ms3: Todo Operations

- **f7** (pending) Add new todo input flow
- **f8** (pending) Space toggles completion
- **f9** (pending) Mark in-progress with `i`

### ms4: Polish, Help, and Integration

- **f10** (pending) Lipgloss styling and layout
- **f11** (pending) Help footer with keybindings
- **f12** (pending) Integration smoke tests
