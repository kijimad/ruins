#!/bin/bash
set -eu

##################################
# Git hooksをセットアップするスクリプト
##################################

cd "$(dirname "$0")"
cd ../

# 定数
HOOKS_DIR="scripts/hooks"

echo "📦 Setting up Git hooks..."
echo "Project root: $PWD"
echo "Hooks directory: $HOOKS_DIR"

# hooksディレクトリが存在することを確認
if [ ! -d "$HOOKS_DIR" ]; then
	echo "❌ Hooks directory not found: $HOOKS_DIR"
	exit 1
fi

# Git設定でhooksディレクトリを指定（プロジェクトレベル）
git config core.hooksPath "$HOOKS_DIR"

echo "✅ Git hooks configured successfully!"
echo ""
echo "Available hooks:"
for hook in "$HOOKS_DIR"/*; do
	if [ -f "$hook" ]; then
		echo "  - $(basename "$hook")"
	fi
done
echo ""
echo "To test: make a commit and watch the pre-commit hook run"
echo "To bypass: use 'git commit --no-verify'"
echo "To disable: run './scripts/remove-hooks.sh'"
