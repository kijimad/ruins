#!/bin/bash
# ブランチで変更したテストのパッケージを反復実行し、フレーキーテストをマージ前に炙り出す。
# 呼び出しは make grill-branch から。xvfb と bwrap は make 側で被せる。
#
# 環境変数:
#   GRILL_BASE  比較の基点。既定 origin/main
#   GRILL_COUNT 反復回数。既定 10
set -eu

base="${GRILL_BASE:-origin/main}"
count="${GRILL_COUNT:-10}"

# 変更のあったテストファイル。git diff の失敗は set -e でそのまま落とす
changed=$(git diff --name-only "${base}...HEAD" -- '*_test.go')

# パッケージへ変換する。削除済みディレクトリとテスト対象外のパッケージは除く。
# grep の不一致は「対象なし」なので失敗にしない
pkgs=$(echo "$changed" |
	xargs -r -n1 dirname | sort -u |
	grep -v -e '^editor-ui/' -e '^oapi$' |
	while read -r d; do [ -d "$d" ] && echo "./$d"; done) || true

if [ -z "$pkgs" ]; then
	echo "grill-branch: ${base} からの変更にテストは無い"
	exit 0
fi

echo "grill-branch: 次のパッケージを -race -count=${count} -shuffle=on で反復する"
echo "$pkgs"

# シャッフル seed は go test が冒頭に出力する。失敗時は -shuffle=<seed> で再現する。
# $pkgs は語分割させるため意図的にクォートしない
# shellcheck disable=SC2086
go test -race -count="$count" -shuffle=on -timeout=60m $pkgs
