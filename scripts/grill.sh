#!/bin/bash
# 全テストを時間予算いっぱい反復実行し、フレーキーテストを炙り出す。夜間のローカル実行用。
# 呼び出しは make grill から。xvfb と bwrap は make 側で被せ、全ラウンドを1つの X セッションで回す。
#
# ラウンドごとに条件を振る:
#   - シャッフル順は毎ラウンド変わる。seed はログに残るので -shuffle=<seed> で再現できる
#   - 3ラウンドに1回 -race を付ける
#   - GOMAXPROCS を 1 / 2 / 全コア と巡回させ、スケジューリング依存を露出させる
#
# 環境変数:
#   GRILL_MINUTES 時間予算(分)。既定 360 で6時間
#   GRILL_LOG_DIR ログ出力先。既定 /tmp/ruins-grill/<開始時刻>
set -u

minutes="${GRILL_MINUTES:-360}"
logdir="${GRILL_LOG_DIR:-/tmp/ruins-grill/$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$logdir"

pkgs=$(go list ./... | grep -v -e /editor-ui/ -e '/oapi$')
deadline=$(($(date +%s) + minutes * 60))
round=0

echo "grill: ${minutes}分の予算で反復する。ログ: $logdir"

while [ "$(date +%s)" -lt "$deadline" ]; do
	round=$((round + 1))
	tag=$(printf 'round-%03d' "$round")

	race=""
	[ $((round % 3)) -eq 0 ] && race="-race"
	case $((round % 3)) in
	1) procs=1 ;;
	2) procs=2 ;;
	*) procs=$(nproc) ;;
	esac

	echo "$tag: race=${race:-off} GOMAXPROCS=$procs"
	# shellcheck disable=SC2086
	if GOMAXPROCS=$procs go test $race -shuffle=on -count=1 -timeout=60m -json $pkgs \
		>"$logdir/$tag.json" 2>"$logdir/$tag.stderr"; then
		rm -f "$logdir/$tag.stderr"
	else
		echo "$tag: FAIL"
	fi
done

echo "== grill 結果: ${round}ラウンド =="
if command -v jq >/dev/null; then
	jq -r 'select(.Action == "fail" and .Test != null) | .Package + " " + .Test' \
		"$logdir"/round-*.json 2>/dev/null | sort | uniq -c | sort -rn | tee "$logdir/summary.txt"
else
	grep -l '"Action":"fail"' "$logdir"/round-*.json | tee "$logdir/summary.txt"
fi
echo "再現: 失敗ラウンドの json から '-test.shuffle' を grep し、go test -shuffle=<seed> で同じ順序を再実行する"

# 1件でも失敗があれば非0で終える
! grep -q '"Action":"fail"' "$logdir"/round-*.json 2>/dev/null || exit 1
