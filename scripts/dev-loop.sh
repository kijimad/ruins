#!/bin/bash
set -eu

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
STATE_FILE="$PROJECT_DIR/.claude-task-state"
DESIGN_DOC="${1:?使い方: ./scripts/dev-loop.sh docs/design/YYYYMMDD_NN.md}"
PROMPT="設計ドキュメント ${DESIGN_DOC} の進捗セクションとgit diffを確認し、未完了の実装タスクを続けて。タスク完了ごとにmake checkを実行し、通ったらcommitすること。同じ系統のエラーで5回失敗したらブロッカーとして進捗セクションに記録し、状態ファイルをblockedにして停止すること。全タスク完了時は状態ファイルをdoneにすること。"

echo "pending" > "$STATE_FILE"

while true; do
  STATE=$(cat "$STATE_FILE")

  case "$STATE" in
    pending)
      echo "running" > "$STATE_FILE"
      cd "$PROJECT_DIR"
      claude -p "$PROMPT" --dangerously-skip-permissions || true

      # Claudeが状態を更新せず終了した場合はpendingに戻す
      [ "$(cat "$STATE_FILE")" = "running" ] && echo "pending" > "$STATE_FILE"
      ;;
    done)
      echo "全タスク完了"
      exit 0
      ;;
    blocked)
      echo "ブロッカー発生。設計ドキュメントの進捗セクションを確認してください"
      exit 1
      ;;
    running)
      echo "pending" > "$STATE_FILE"
      ;;
  esac

  sleep 60
done
