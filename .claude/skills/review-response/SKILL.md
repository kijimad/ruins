---
name: review-response
description: PR のレビュー指摘に対応する手順。指摘ごとに対応するか判断し、対応した項目と対応しない項目を理由付きで PR の description に追記する。「レビュー対応せよ」と言われたときに使う
---

# レビュー対応

## 1. 最新のレビューを読む

- 読む対象は3つある。claude-review の issue コメント、人の review、行単位の review comment。どれも件数が多いと既定の30件で切れるので `--paginate` を付ける。

  ```sh
  gh api --paginate repos/<owner>/<repo>/issues/<PR>/comments --jq '.[] | select(.user.login=="claude[bot]") | .updated_at, .body'
  gh api --paginate repos/<owner>/<repo>/pulls/<PR>/reviews --jq '.[] | .user.login, .state, .body'
  gh api --paginate repos/<owner>/<repo>/pulls/<PR>/comments --jq '.[] | .path, .line, .body'
  ```

- claude-review は PR に1つのコメントを置き、走るたびに上書きする。本文の View job の run が今の HEAD に対するものかを、run の headSha で確かめる。push の後に走り終わる前の古いコメントを読まないためである。

  ```sh
  gh run view <run_id> --json headSha --jq .headSha   # git rev-parse HEAD と比べる
  ```

- CI の失敗も見る。`gh pr checks <PR>` で落ちたジョブのログを読み、変更による回帰か、既知のフレークかを切り分ける。

## 2. 指摘ごとに判断する

- 中身で判断する。同じ指摘が繰り返されたことは、直す理由にならない。前の回に対応しないとした指摘は、前回と同じ根拠で対応しない。
- 指摘の前提をコードで確かめる。レビューの読み違いや計算の誤りは、対応しない理由として根拠を添える。

## 3. 直す

- 対応する項目を直し、`make check` を終了コードで確かめてからコミットする。

## 4. PR の description に追記する

- push の前に description を書き換える。push で claude-review が走り直し、そのとき description も読むので、判断の経緯を踏まえたレビューになる。
- description に `## レビュー対応` の節を置き、レビューのたびに表へ行を足していく。前の回の行は消さない。
- 対応した項目も、対応しない項目も、理由付きで書く。セルに `|` を書くときは `\|` とエスケープするか、文に言い換える。

  ```markdown
  ## レビュー対応

  | 指摘 | 対応 | 理由 |
  |---|---|---|
  | 屋内が屋外より明るくなりうる | 直した | 日照で頭打ちにし、テストに行を足した |
  | 底上げの倍率まで vision で計算する | 対応しない | 描画の調整値が視界の計算へ漏れる |
  ```

- 変更点・採用しなかった代替・検証・リスクの節に関わる内容は、その節にも反映する。
- GitHub へのコメントでは返さない。対応の記録は description だけに残す。
- 書き換えは `gh pr view <PR> --json body --jq .body` で今の本文を取り、編集して `gh pr edit <PR> --body-file` で戻す。本文のファイルは `mktemp` で worktree の外に作り、誤ってコミットしないようにする。ヒアドキュメントで書くときは `<<'EOF'` と引用符で囲み、バッククォートがシェルに解釈されないようにする。

## 5. push する

- description を書き換えた後に push する。
