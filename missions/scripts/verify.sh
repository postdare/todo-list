#!/usr/bin/env bash
# verify.sh — Mission 2026-08-31-todo 验收脚本
# 用法: ./missions/scripts/verify.sh <branch>
# 每个分支幂等、可重复运行;退出码 0=通过,非 0=失败,2=未知分支。
set -euo pipefail

BRANCH="${1:-}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

case "$BRANCH" in
  build)
    # AC4:全仓可编译
    go build ./...
    ;;

  fmt)
    # AC4:全部 Go 源码通过 gofmt
    out="$(gofmt -l .)"
    if [ -n "$out" ]; then
      echo "以下文件未通过 gofmt:" >&2
      echo "$out" >&2
      exit 1
    fi
    ;;

  test-unit)
    # AC1/AC2/AC3:编辑模式(保存/取消/字段保持)、列宽与渲染契约(14 汉字完整显示、无重叠、样式可区分)
    go test ./internal/...
    ;;

  test-e2e)
    # AC1/AC2/AC4:PTY 端到端 —— 编辑流程持久化(JSON 更新且 id/status/created_at 不变)、长标题完整显示、看板/描述/持久化回归
    go test ./cmd/...
    ;;

  contract)
    # AC4:回归契约 —— 描述字段的 JSON 键仍为 description(上一功能不回归)
    grep -q 'json:"description"' internal/todo/todo.go
    ;;

  *)
    echo "未知分支: $BRANCH" >&2
    exit 2
    ;;
esac
