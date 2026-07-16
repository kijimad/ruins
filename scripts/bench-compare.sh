#!/usr/bin/env bash
# ベース(既定 origin/main)と現在の作業ツリーのベンチマークを比較する。
# - benchstat の比較表を標準出力に出す（PRのstickyコメント用）
# - sec/op が THRESHOLD 倍(既定2.0=2倍)以上悪化したベンチがあれば非ゼロ終了する（破壊的悪化の門番）
#
# 使い方: BASE=origin/main make bench-compare
# ベースにベンチが存在しない場合（新規ベンチ）は比較対象なしとして扱い、門番は通過する。
set -euo pipefail

BASE="${BASE:-origin/main}"
BENCH="${BENCH:-BenchmarkProcessAll|BenchmarkRestoreAllActionPoints}"
PKGS="${PKGS:-./internal/aiinput/ ./internal/world/query/}"
COUNT="${COUNT:-6}"
BENCHTIME="${BENCHTIME:-100ms}"
THRESHOLD="${THRESHOLD:-2.0}"

OUT_DIR="$(mktemp -d)"
OLD="${OUT_DIR}/base.txt"
NEW="${OUT_DIR}/pr.txt"
WORKTREE="${OUT_DIR}/base-src"

# ebiten のヘッドレス実行のため bwrap(利用可能なら) + xvfb でラップする
BWRAP=$(bwrap --dev-bind / / --tmpfs /dev/input -- true 2>/dev/null && echo "bwrap --dev-bind / / --tmpfs /dev/input --" || true)

run_bench() { # $1=作業ディレクトリ $2=出力ファイル
	(cd "$1" && RUINS_LOG_LEVEL=ignore $BWRAP xvfb-run -a \
		go test $PKGS -run '^$' -bench="$BENCH" -benchmem -count="$COUNT" -benchtime="$BENCHTIME") \
		>"$2" 2>/dev/null || true
}

cleanup() {
	git worktree remove --force "$WORKTREE" 2>/dev/null || true
	rm -rf "$OUT_DIR"
}
trap cleanup EXIT

echo "==> 現在(PR)のベンチを実行" >&2
run_bench "$(pwd)" "$NEW"

echo "==> ベース($BASE)のベンチを実行" >&2
if git rev-parse --verify --quiet "$BASE" >/dev/null; then
	git worktree add --detach "$WORKTREE" "$BASE" >/dev/null 2>&1
	run_bench "$WORKTREE" "$OLD"
else
	echo "警告: ベース参照 '$BASE' が見つからないため比較をスキップします" >&2
	: >"$OLD"
fi

echo ""
echo "### ベンチマーク比較 (base=$BASE)"
echo '```'
benchstat "$OLD" "$NEW" 2>/dev/null || echo "(benchstat の比較を生成できませんでした)"
echo '```'
echo ""

# 門番: sec/op の悪化倍率が THRESHOLD 以上のベンチを検出する。
# benchstat の "vs base" 列は低サンプルだと "~" になるため、old/new の生の平均から自前で比を計算する。
REGRESSIONS=$(benchstat -format csv "$OLD" "$NEW" 2>/dev/null | awk -F, -v T="$THRESHOLD" '
	$1=="" && $2!="" { metric=$2; next }                       # メトリクスのヘッダ行
	$1!="" && $1!="geomean" && metric=="sec/op" && ($2+0==$2) && ($4+0==$4) && $2>0 {
		ratio = $4 / $2
		if (ratio >= T) printf "- %s: x%.2f (%.3g -> %.3g sec/op)\n", $1, ratio, $2, $4
	}
')

if [ -n "$REGRESSIONS" ]; then
	echo "🚨 破壊的悪化を検出 (sec/op が ${THRESHOLD}倍以上):"
	echo "$REGRESSIONS"
	exit 1
fi

echo "✅ ${THRESHOLD}倍以上の悪化なし"
