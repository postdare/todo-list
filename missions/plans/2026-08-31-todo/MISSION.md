# Mission: 2026-08-31-todo

tier: complex
goal: 为看板 todo 应用补齐编辑功能(选中任务可修改标题与描述,保持 id/status/created_at 不变),并优化布局样式:列宽统一不重叠、行内去冗余 padding、竖向更紧凑,使内容区在 100 列终端下能完整显示 14 个汉字;描述契约与全部既有行为不回归。

## Frozen Acceptance Criteria

> 本节只读。修改必须走 L3 升级,并以 `mission-escalate(L3):` 前缀单独提交。

- AC1 (verify: `test-unit`): 编辑功能:选中任务按 e 进入编辑模式,标题与描述两段预填可修改,Enter 保存(经 store.Update,id/status/created_at 保持不变),Esc 任一阶段取消不落盘,空标题不保存;保存后 JSON 持久化反映编辑结果。
- AC2 (verify: `test-e2e`): 内容区扩容:三列总宽统一为 (width-2)/3(100 列下每列 32 格),行内无横向 padding,文字 fit 上限 = 列宽-前缀;100 列终端下 14 个汉字的标题完整显示不截断;标题与看板之间仅 1 个空行。
- AC3 (verify: `test-unit`): 样式与渲染:三栏并排渲染且列宽一致、行宽不超过分配列宽(无重叠溢出);焦点列头与选中行使用可区分的背景色(焦点列头青色底、选中行 252 底);描述行以斜体缩进显示;渲染结构为 标题→列头行(含三状态)→内容→footer。
- AC4 (verify: `contract`): 回归与构建质量:描述 JSON 契约(description 键)不回归,全仓编译、gofmt 合规,全部单测与 PTY 端到端测试通过。

## Milestones

- [ ] M1 编辑功能(实现 → 单测) → [M1.md](./M1.md)
- [ ] M2 布局与样式优化(内容区扩容 → 渲染契约) → [M2.md](./M2.md)
- [ ] M3 端到端与回归 → [M3.md](./M3.md)

> 任务明细按里程碑分文件(I8),到哪个里程碑读哪个文件。

```mission
{
  "missionId": "2026-08-31-todo",
  "tier": "complex",
  "goal": "为看板 todo 应用补齐编辑功能(选中任务可修改标题与描述,保持 id/status/created_at 不变),并优化布局样式:列宽统一不重叠、行内去冗余 padding、竖向更紧凑,使内容区在 100 列终端下能完整显示 14 个汉字;描述契约与全部既有行为不回归。",
  "acceptanceCriteria": [
    {
      "id": "AC1",
      "text": "编辑功能:选中任务按 e 进入编辑模式,标题与描述两段预填可修改,Enter 保存(经 store.Update,id/status/created_at 保持不变),Esc 任一阶段取消不落盘,空标题不保存;保存后 JSON 持久化反映编辑结果。",
      "verify": "test-unit"
    },
    {
      "id": "AC2",
      "text": "内容区扩容:三列总宽统一为 (width-2)/3(100 列下每列 32 格),行内无横向 padding,文字 fit 上限 = 列宽-前缀;100 列终端下 14 个汉字的标题完整显示不截断;标题与看板之间仅 1 个空行。",
      "verify": "test-e2e"
    },
    {
      "id": "AC3",
      "text": "样式与渲染:三栏并排渲染且列宽一致、行宽不超过分配列宽(无重叠溢出);焦点列头与选中行使用可区分的背景色(焦点列头青色底、选中行 252 底);描述行以斜体缩进显示;渲染结构为 标题→列头行(含三状态)→内容→footer。",
      "verify": "test-unit"
    },
    {
      "id": "AC4",
      "text": "回归与构建质量:描述 JSON 契约(description 键)不回归,全仓编译、gofmt 合规,全部单测与 PTY 端到端测试通过。",
      "verify": "contract"
    }
  ],
  "milestones": [
    {
      "id": "M1",
      "title": "编辑功能(实现 → 单测)",
      "tasks": [
        {
          "id": "T1",
          "title": "编辑模式实现:非输入态对选中任务按 e 进入编辑(空栏无操作);复用两段式输入机制,标题与描述预填当前值;Enter 保存经 store.Update 且保留 id/status/created_at;Esc 任一阶段取消不落盘;空标题不保存;footer 增加 e 编辑 提示,编辑中提示文案为 编辑标题/编辑描述。",
          "verify": [
            "test-unit"
          ]
        },
        {
          "id": "T2",
          "title": "编辑单测:保存后 title/description 更新且 id/status/created_at 不变;Esc 取消不落盘;仅改描述时标题不变;编辑不改变任务所在栏;空标题保存被忽略。",
          "verify": [
            "test-unit"
          ]
        }
      ]
    },
    {
      "id": "M2",
      "title": "布局与样式优化(内容区扩容 → 渲染契约)",
      "tasks": [
        {
          "id": "T3",
          "title": "内容区扩容与样式:列总宽统一 = (width-2)/3,行内去横向 padding,文字 fit 上限 = 列宽-前缀(100 列下可完整显示 14 汉字);标题与看板之间仅 1 空行;描述行斜体缩进;焦点列头青色底/选中行 252 底保持可区分;确保三列并排无重叠溢出。",
          "verify": [
            "test-unit"
          ]
        },
        {
          "id": "T4",
          "title": "渲染契约单测:columnWidth(100) 断言;View 渲染结构(非空行顺序:标题→含三状态名的列头行→内容→footer);14 汉字标题完整渲染不截断;行宽 ≤ 分配列宽(无重叠);描述行样式断言。",
          "verify": [
            "test-unit"
          ]
        }
      ]
    },
    {
      "id": "M3",
      "title": "端到端与回归",
      "tasks": [
        {
          "id": "T5",
          "title": "PTY 端到端扩展与回归:新增编辑流程 e2e(新建→e 编辑标题/描述→保存→JSON 中 title/description 更新且 id/status/created_at 不变;e 后 Esc 取消不落盘);新增 14 汉字长标题完整显示 e2e;既有看板/两段式/描述/持久化 e2e 保持通过;build/fmt/contract 全绿。",
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
  "verifyScript": "#!/usr/bin/env bash\n# verify.sh — Mission 2026-08-31-todo 验收脚本\n# 用法: ./missions/scripts/verify.sh <branch>\n# 每个分支幂等、可重复运行;退出码 0=通过,非 0=失败,2=未知分支。\nset -euo pipefail\n\nBRANCH=\"${1:-}\"\nROOT=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")/../..\" && pwd)\"\ncd \"$ROOT\"\n\ncase \"$BRANCH\" in\n  build)\n    # AC4:全仓可编译\n    go build ./...\n    ;;\n\n  fmt)\n    # AC4:全部 Go 源码通过 gofmt\n    out=\"$(gofmt -l .)\"\n    if [ -n \"$out\" ]; then\n      echo \"以下文件未通过 gofmt:\" >&2\n      echo \"$out\" >&2\n      exit 1\n    fi\n    ;;\n\n  test-unit)\n    # AC1/AC2/AC3:编辑模式(保存/取消/字段保持)、列宽与渲染契约(14 汉字完整显示、无重叠、样式可区分)\n    go test ./internal/...\n    ;;\n\n  test-e2e)\n    # AC1/AC2/AC4:PTY 端到端 —— 编辑流程持久化(JSON 更新且 id/status/created_at 不变)、长标题完整显示、看板/描述/持久化回归\n    go test ./cmd/...\n    ;;\n\n  contract)\n    # AC4:回归契约 —— 描述字段的 JSON 键仍为 description(上一功能不回归)\n    grep -q 'json:\"description\"' internal/todo/todo.go\n    ;;\n\n  *)\n    echo \"未知分支: $BRANCH\" >&2\n    exit 2\n    ;;\nesac\n",
  "createdAt": 1788196740618
}
```
