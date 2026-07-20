---
name: ruins-review
description: ruins 固有のレビュー観点チェックリスト。Ark の strict semantics、serde 安全性、生成物、型アサーションなど繰り返し出た論点をまとめる。コードレビューや自己点検のときに使う
---

# ruins レビュー観点

**徹底的に批判的にレビューする。** 以下はこのプロジェクトのレビューで繰り返し登場した論点。

## Ark ECS の strict semantics（すべて panic する）

- 死亡エンティティへの Get/Has/Remove、不在コンポーネントへの Get/Set、二重 Add はすべて panic する。`world.ECS.Alive()` / `Has()` で事前に弾くか `gc.Upsert` を使う。
- `Add` は値をアーキタイプ領域へ**コピー**する（呼び出し側ポインタ ≠ 格納先）。`Get` はその領域へのポインタを返し、**構造変更で無効化**される。構造変更後は再取得する。値をコピーして書き換えても格納先には反映されない。
- クエリ反復中はワールドがロックされる。反復中に構造変更（生成/削除/Add/Remove）してはいけない。collect-then-mutate（一旦集めてからループ外で変更）する。

## serde（丸ごと保存）

- 保存対象コンポーネントは serde 安全であること: interface を持たない、map のキーは string。struct キーの map は `json:"-"` + ロード時再構築。
- 一時・実行時状態は `skipComponents()` に入れる。
- エンティティ参照は ID 空間の保存で整合させる。

## コード規約

- 未チェックの型アサーションを書かない（forcetypeassert 有効）。comma-ok にするか、不変条件違反なら panic させる。
- コンポーネント追加は登録表 + `make generate`（add-component skill 参照）。`components_gen.go` を手編集しない。
- 不要なゲッター/セッターを作らない。コンストラクタは型ごとに最大1つ（必要ならオプションパターン）。
- 生成ファイル（`*_gen.go`）を sed 等で直接書き換えない。生成元を直して再生成する。

## 列挙の網羅（散在スイッチ・クエリ）

ある概念が**複数のスイッチ/クエリに散在**し、1種別足すと全箇所に足す必要がある。1つでも漏れると、発動しない・表示されないといった silent な不具合になり、手動プレイで初めて発覚しがち。

**機械化を最優先する。** enum を switch する箇所は **default を置かず** exhaustive linter に全 case を強制させる。`.golangci.yml` は `default-signifies-exhaustive: true`（default があれば網羅とみなす）なので、**強制したい switch は default を消す**のが要点。値を返す switch は switch の後に fall-through 用の一文を置く:

- **内部の信頼できる値**を switch する場合は末尾に `panic("未知の X: " + string(k))`。Go 版の never 相当。
- **raw/save 由来の未信頼な値**を switch する場合は panic 禁止。末尾でゼロ値など graceful なフォールバックを返し、呼び出し側の検証に委ねる。`Config()` はこの型で、末尾 `return InteractionConfig{}`。exhaustive は既知種別の網羅を強制しつつ、未知入力は末尾へ落ちる。

これで新種別の対応漏れが `make lint` で止まる。

- **`InteractionKind` は全スイッチ exhaustive 強制済み**: `interactable.go` の `Config()`、`states/action_handlers.go` の `getInteractionActions`、`activity/execute_interaction.go`、`activity/player_actions.go` の足元ログ、いずれも default を持たない。新種別を足すと4箇所すべてで lint が漏れを検知する。手動照合は不要。
- **default を残すのが正しい enum もある**: `inputmapper.ActionID` の入力ハンドラは多数のアクションから一部だけ処理し残りを default で流すのが正常。exhaustive 強制は不適切なので default を残す。「全 case 列挙が意味を持つドメイン enum」だけ default を外す。
- **ステージ跨ぎクエリ（Phase 7 共存方式）**: 退避中ステージのエンティティを含みうるクエリは生の `ecs.NewFilterN` でなく `query.ActiveFilter` を使う。`Suspended` 除外を1箇所に集約している。linter 強制は無いので `grep -rn "ecs.NewFilter" internal/ | grep -iE "GridElement|Door|SoloAI|Interactable"` 等で洗う。座標検索・破壊的操作（一括削除/平行移動）の漏れが特に致命的。
- 一般方針: 新しい enum を足すなら、全 case 列挙が意味を持つなら扱う switch から default を外して exhaustive に強制させる。多数から一部だけ処理する enum は default を残し、追加時に grep で照合する。

## 参照

- 設計・コード規約の詳細は CLAUDE.md。
- コンポーネント追加の手順は add-component skill。
