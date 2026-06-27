#!/bin/bash
set -eu

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
STATE_FILE="$PROJECT_DIR/.claude-task-state"
DESIGN_DIR="$PROJECT_DIR/docs/design"
cd "$PROJECT_DIR"

# 未完了タスクがある設計ドキュメントを探す
find_next_design_doc() {
  for doc in $(ls "$DESIGN_DIR"/*.md 2>/dev/null | sort); do
    if grep -q '^\- \[ \]' "$doc"; then
      echo "$doc"
      return
    fi
  done
  echo ""
}

# 起動時は常にpendingにリセットする。blocked/doneは前回セッションの終了状態であり、
# スクリプトを再起動する時点で人間が状況を確認済みと判断する
echo "pending" > "$STATE_FILE"

while true; do
  STATE=$(cat "$STATE_FILE")

  case "$STATE" in
    pending)
      DESIGN_DOC=$(find_next_design_doc)
      if [ -z "$DESIGN_DOC" ]; then
        echo "未完了の設計ドキュメントなし"
        exit 0
      fi

      REL_PATH="${DESIGN_DOC#"$PROJECT_DIR"/}"
      TASK=$(grep -m1 '^\- \[ \]' "$DESIGN_DOC" | sed 's/^- \[ \] //')
      echo "実装中: $REL_PATH ($TASK)"
      PROMPT="設計ドキュメント ${REL_PATH} の進捗セクションとgit diffを確認し、未完了の実装タスクを続けて。タスク完了ごとにmake checkを実行し、通ったらcommitすること。同じ系統のエラーで5回失敗したらブロッカーとして進捗セクションに記録し、状態ファイルをblockedにして停止すること。全タスク完了時は状態ファイルをdoneにすること。"

      echo "running" > "$STATE_FILE"
      if ! claude -p "$PROMPT" --dangerously-skip-permissions; then
        echo "警告: Claudeが異常終了した" >&2
      fi

      # Claudeが状態を更新せず終了した場合はpendingに戻す
      [ "$(cat "$STATE_FILE")" = "running" ] && echo "pending" > "$STATE_FILE"
      sleep 60
      ;;
    done)
      # 現在のドキュメントが完了。次のドキュメントを探す
      echo "pending" > "$STATE_FILE"
      ;;
    blocked)
      echo "ブロッカー発生。設計ドキュメントの進捗セクションを確認してください"
      exit 1
      ;;
    running)
      # 前回セッションがクラッシュした可能性がある
      echo "前回セッションが異常終了。再開する"
      echo "pending" > "$STATE_FILE"
      ;;
  esac
done
