# Mission: 2026-08-31-todolist

tier: standard
goal: todolist 支持新增描述:新建任务时可录入描述(可选),描述随任务持久化并在列表视图中可见;旧数据(无 description 键)保持兼容。

## Frozen Acceptance Criteria

> 本节只读。修改必须走 L3 升级,并以 `mission-escalate(L3):` 前缀单独提交。

- AC1 (verify: `contract`): 数据模型与持久化契约:Todo 领域模型新增 Description 字段,JSON 键为 description;含描述的任务可完整持久化并读回;不含 description 键的旧 todos.json 仍能正常加载(向后兼容,无需迁移)。
- AC2 (verify: `test-unit`): TUI 两段式新增流程:按 n 新建时先输入标题,Enter 进入描述输入(描述可留空,直接 Enter 保存);保存后任务同时含标题与描述,有描述的任务在列表视图中可见其描述;Esc 在标题或描述任一阶段取消且不落盘。
- AC3 (verify: `test-e2e`): 端到端与回归:PTY 端到端测试证明通过真实 TUI 输入标题+描述后,todos.json 中含 description 键且值正确、界面渲染出描述;全仓编译、gofmt 合规、全部单测与端到端测试通过。

## Tasks

- [ ] T1 领域模型:在 internal/todo/todo.go 的 Todo 结构体新增 Description 字段,JSON tag 为 description。 (verify: `contract`, `test-unit`)
- [ ] T2 契约测试:更新 internal/todo 与 internal/storage 单测,覆盖 description 的序列化/反序列化、持久化读回、旧数据无 description 键时加载为空(向后兼容)。 (verify: `test-unit`)
- [ ] T3 TUI 两段式新增与展示:新增流程改为「标题 → 描述(可跳过)」两段输入,Enter 进入下一段、末段 Enter 保存、Esc 任意阶段取消;列表中对有描述的任务渲染描述(次要行或次要文本);同步更新受影响的既有 internal/tui 测试(如 TestAddTodoViaInput 需多按一次 Enter)。 (verify: `test-unit`, `test-e2e`)
- [ ] T4 端到端与回归:扩展 cmd/todo-list/main_test.go 的 PTY 测试,断言输入标题+描述后 todos.json 含 description 键且值正确、界面可见描述;确保 go build ./... 与 gofmt 全绿。 (verify: `build`, `fmt`, `test-e2e`)

```mission
{
  "missionId": "2026-08-31-todolist",
  "tier": "standard",
  "goal": "todolist 支持新增描述:新建任务时可录入描述(可选),描述随任务持久化并在列表视图中可见;旧数据(无 description 键)保持兼容。",
  "acceptanceCriteria": [
    {
      "id": "AC1",
      "text": "数据模型与持久化契约:Todo 领域模型新增 Description 字段,JSON 键为 description;含描述的任务可完整持久化并读回;不含 description 键的旧 todos.json 仍能正常加载(向后兼容,无需迁移)。",
      "verify": "contract"
    },
    {
      "id": "AC2",
      "text": "TUI 两段式新增流程:按 n 新建时先输入标题,Enter 进入描述输入(描述可留空,直接 Enter 保存);保存后任务同时含标题与描述,有描述的任务在列表视图中可见其描述;Esc 在标题或描述任一阶段取消且不落盘。",
      "verify": "test-unit"
    },
    {
      "id": "AC3",
      "text": "端到端与回归:PTY 端到端测试证明通过真实 TUI 输入标题+描述后,todos.json 中含 description 键且值正确、界面渲染出描述;全仓编译、gofmt 合规、全部单测与端到端测试通过。",
      "verify": "test-e2e"
    }
  ],
  "milestones": [
    {
      "id": "M1",
      "title": "描述功能落地(领域模型 → 契约测试 → TUI 交互 → 端到端回归)",
      "tasks": [
        {
          "id": "T1",
          "title": "领域模型:在 internal/todo/todo.go 的 Todo 结构体新增 Description 字段,JSON tag 为 description。",
          "verify": [
            "contract",
            "test-unit"
          ]
        },
        {
          "id": "T2",
          "title": "契约测试:更新 internal/todo 与 internal/storage 单测,覆盖 description 的序列化/反序列化、持久化读回、旧数据无 description 键时加载为空(向后兼容)。",
          "verify": [
            "test-unit"
          ]
        },
        {
          "id": "T3",
          "title": "TUI 两段式新增与展示:新增流程改为「标题 → 描述(可跳过)」两段输入,Enter 进入下一段、末段 Enter 保存、Esc 任意阶段取消;列表中对有描述的任务渲染描述(次要行或次要文本);同步更新受影响的既有 internal/tui 测试(如 TestAddTodoViaInput 需多按一次 Enter)。",
          "verify": [
            "test-unit",
            "test-e2e"
          ]
        },
        {
          "id": "T4",
          "title": "端到端与回归:扩展 cmd/todo-list/main_test.go 的 PTY 测试,断言输入标题+描述后 todos.json 含 description 键且值正确、界面可见描述;确保 go build ./... 与 gofmt 全绿。",
          "verify": [
            "build",
            "fmt",
            "test-e2e"
          ]
        }
      ]
    }
  ],
  "verifyScript": "#!/usr/bin/env bash\n# verify.sh — Mission 2026-08-31-todolist 验收脚本\n# 用法: ./missions/scripts/verify.sh <branch>\n# 每个分支幂等、可重复运行;退出码 0=通过,非 0=失败,2=未知分支。\nset -euo pipefail\n\nBRANCH=\"${1:-}\"\nROOT=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")/../..\" && pwd)\"\ncd \"$ROOT\"\n\ncase \"$BRANCH\" in\n  build)\n    # AC3:全仓可编译\n    go build ./...\n    ;;\n\n  fmt)\n    # AC3:全部 Go 源码通过 gofmt\n    out=\"$(gofmt -l .)\"\n    if [ -n \"$out\" ]; then\n      echo \"以下文件未通过 gofmt:\" >&2\n      echo \"$out\" >&2\n      exit 1\n    fi\n    ;;\n\n  test-unit)\n    # AC1/AC2:领域模型、存储、TUI 单测(含 description 契约与两段式录入)\n    go test ./internal/...\n    ;;\n\n  test-e2e)\n    # AC2/AC3:PTY 端到端 —— 真实 TUI 录入标题+描述、JSON 持久化含 description 键、界面渲染描述\n    go test ./cmd/...\n    ;;\n\n  contract)\n    # AC1:领域模型契约 —— Description 字段的 JSON 键必须是 description\n    grep -q 'json:\"description\"' internal/todo/todo.go\n    ;;\n\n  *)\n    echo \"未知分支: $BRANCH\" >&2\n    exit 2\n    ;;\nesac\n",
  "createdAt": 1788195779035
}
```
