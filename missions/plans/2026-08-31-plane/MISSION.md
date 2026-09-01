# Mission: 2026-08-31-plane

tier: standard
goal: 看板视图:将 tab 式切换 UI 改造为看板布局,未开始/进行中/已完成 三个板块(栏)在同一画面并排同时展示,任何时刻三栏都可见;←/→ 仅移动光标栏(不隐藏任何栏),↑/↓ 栏内移动;n/i/Space 新建与状态流转、两段式标题/描述输入、持久化契约全部保留。

## Frozen Acceptance Criteria

> 本节只读。修改必须走 L3 升级,并以 `mission-escalate(L3):` 前缀单独提交。

- AC1 (verify: `test-unit`): 三栏同屏可见(非切换):启动后不按任何键,未开始/进行中/已完成 三个栏(列头+各栏内容)同时渲染在同一画面;任何时刻三栏都可见,←/→ 只移动焦点栏,不隐藏任何栏。
- AC2 (verify: `test-e2e`): 看板交互:↑/↓ 在栏内移动、←/→ 在栏间移动焦点;n 两段式新建(标题/描述)落 未开始 栏并选中新任务;i 将选中任务移入 进行中 栏;Space 在 已完成↔未开始 间切换;跨栏移动后持久化(JSON status)与光标状态正确。
- AC3 (verify: `contract`): 回归与构建质量:描述 JSON 契约(description 键)不回归,全仓编译、gofmt 合规,全部单测与 PTY 端到端测试通过。

## Tasks

- [ ] T1 状态模型与三栏渲染:Model 以 activeCol(0..2) 取代 activeTab,selected 语义改为焦点栏内索引;View 删除 tab bar,改为三栏并排渲染(列头+各栏任务+空栏提示,三个状态名恒可见),选中行仅焦点栏高亮;←/→ 栏间移动、↑/↓ 栏内移动,footer 文案同步更新;同步重写 internal/tui 中依赖 tab 切换渲染语义的既有测试(如 TestSpaceTogglesDone/TestMarkInProgress/TestNavigationSelection 中“某 tab 不再显示”类断言)。 (verify: `test-unit`)
- [ ] T2 操作语义迁移:commitAdd 保存后光标落 未开始 栏并选中新任务;toggleDone/markInProgress 状态变更导致任务跨栏后 selected 在所在栏内 clamp;两段式标题/描述输入流程保留;新增单测:View 同时含三个状态名(三栏同屏)、栏间移动、跨栏后光标状态。 (verify: `test-unit`)
- [ ] T3 端到端与回归:改造/新增 cmd PTY 测试——启动后不按任何键即断言三个列头同屏可见;看板交互流程(新建→进行中→完成→重开)的界面与 JSON 持久化断言(status 与 description 键);原 tab 切换相关断言适配;build/fmt/contract 全绿。 (verify: `test-e2e`, `build`, `fmt`, `contract`)

```mission
{
  "missionId": "2026-08-31-plane",
  "tier": "standard",
  "goal": "看板视图:将 tab 式切换 UI 改造为看板布局,未开始/进行中/已完成 三个板块(栏)在同一画面并排同时展示,任何时刻三栏都可见;←/→ 仅移动光标栏(不隐藏任何栏),↑/↓ 栏内移动;n/i/Space 新建与状态流转、两段式标题/描述输入、持久化契约全部保留。",
  "acceptanceCriteria": [
    {
      "id": "AC1",
      "text": "三栏同屏可见(非切换):启动后不按任何键,未开始/进行中/已完成 三个栏(列头+各栏内容)同时渲染在同一画面;任何时刻三栏都可见,←/→ 只移动焦点栏,不隐藏任何栏。",
      "verify": "test-unit"
    },
    {
      "id": "AC2",
      "text": "看板交互:↑/↓ 在栏内移动、←/→ 在栏间移动焦点;n 两段式新建(标题/描述)落 未开始 栏并选中新任务;i 将选中任务移入 进行中 栏;Space 在 已完成↔未开始 间切换;跨栏移动后持久化(JSON status)与光标状态正确。",
      "verify": "test-e2e"
    },
    {
      "id": "AC3",
      "text": "回归与构建质量:描述 JSON 契约(description 键)不回归,全仓编译、gofmt 合规,全部单测与 PTY 端到端测试通过。",
      "verify": "contract"
    }
  ],
  "milestones": [
    {
      "id": "M1",
      "title": "看板化改造(模型与渲染 → 操作迁移 → 端到端回归)",
      "tasks": [
        {
          "id": "T1",
          "title": "状态模型与三栏渲染:Model 以 activeCol(0..2) 取代 activeTab,selected 语义改为焦点栏内索引;View 删除 tab bar,改为三栏并排渲染(列头+各栏任务+空栏提示,三个状态名恒可见),选中行仅焦点栏高亮;←/→ 栏间移动、↑/↓ 栏内移动,footer 文案同步更新;同步重写 internal/tui 中依赖 tab 切换渲染语义的既有测试(如 TestSpaceTogglesDone/TestMarkInProgress/TestNavigationSelection 中“某 tab 不再显示”类断言)。",
          "verify": [
            "test-unit"
          ]
        },
        {
          "id": "T2",
          "title": "操作语义迁移:commitAdd 保存后光标落 未开始 栏并选中新任务;toggleDone/markInProgress 状态变更导致任务跨栏后 selected 在所在栏内 clamp;两段式标题/描述输入流程保留;新增单测:View 同时含三个状态名(三栏同屏)、栏间移动、跨栏后光标状态。",
          "verify": [
            "test-unit"
          ]
        },
        {
          "id": "T3",
          "title": "端到端与回归:改造/新增 cmd PTY 测试——启动后不按任何键即断言三个列头同屏可见;看板交互流程(新建→进行中→完成→重开)的界面与 JSON 持久化断言(status 与 description 键);原 tab 切换相关断言适配;build/fmt/contract 全绿。",
          "verify": [
            "test-e2e",
            "build",
            "fmt",
            "contract"
          ]
        }
      ]
    }
  ],
  "verifyScript": "#!/usr/bin/env bash\n# verify.sh — Mission 2026-08-31-plane 验收脚本\n# 用法: ./missions/scripts/verify.sh <branch>\n# 每个分支幂等、可重复运行;退出码 0=通过,非 0=失败,2=未知分支。\nset -euo pipefail\n\nBRANCH=\"${1:-}\"\nROOT=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")/../..\" && pwd)\"\ncd \"$ROOT\"\n\ncase \"$BRANCH\" in\n  build)\n    # AC3:全仓可编译\n    go build ./...\n    ;;\n\n  fmt)\n    # AC3:全部 Go 源码通过 gofmt\n    out=\"$(gofmt -l .)\"\n    if [ -n \"$out\" ]; then\n      echo \"以下文件未通过 gofmt:\" >&2\n      echo \"$out\" >&2\n      exit 1\n    fi\n    ;;\n\n  test-unit)\n    # AC1/AC2:单测 —— 三栏同屏可见(View 同时含三个状态名)、栏间/栏内导航、跨栏后光标状态、两段式输入与描述契约\n    go test ./internal/...\n    ;;\n\n  test-e2e)\n    # AC2/AC3:PTY 端到端 —— 启动即三个列头同屏可见(无按键)、看板交互全流程、JSON 持久化含 status 与 description\n    go test ./cmd/...\n    ;;\n\n  contract)\n    # AC3:回归契约 —— 描述字段的 JSON 键仍为 description(上一功能不回归)\n    grep -q 'json:\"description\"' internal/todo/todo.go\n    ;;\n\n  *)\n    echo \"未知分支: $BRANCH\" >&2\n    exit 2\n    ;;\nesac\n",
  "createdAt": 1788196201186
}
```
