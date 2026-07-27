---
name: post-merge
description: PRがマージされた後の後片付けを行う。worktreeの破棄とコード知識グラフの更新をするときに使う
---

# マージ後の後片付け

PR がマージされたら以下を順に行う。「マージした」と言われたら着手前にこの手順を実行する。

## 1. 本体を main に追随させる

リポジトリ本体は常に main を checkout したままにする。

```sh
cd <リポジトリ本体> && git pull
```

## 2. worktree とブランチを破棄する

```sh
git worktree remove .claude/worktrees/<name>
git branch -d <branch>
```

- 未コミット変更が残っていないか `git -C <worktree> status` で確認してから消す。
- 次のタスクは新しい worktree を切って始める。

## 3. コード知識グラフを更新する

正のグラフは本体ルートの `graphify-out/` に1つだけ持つ。グラフは main 時点のコードの姿を表す。

- main 更新後に `/graphify internal --update` を本体ルートで実行する。差分ファイルのみ AST 再抽出されるので数十秒で済む。閲覧は `graphify-out/graph.html` を直接開く。

## 注意

- worktree 内でグラフを構築しない。構造の質問は本体ルートのグラフに `/graphify query` する。
- 大規模リファクタ中に作業ブランチのグラフが欲しい場合のみ worktree 内に使い捨て構築し、worktree ごと捨てる。
