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

ある概念が**複数のスイッチ/クエリに散在**し、1種別足すと全箇所に足す必要がある。Go の `exhaustive` linter は default 付き switch を検査しないので、漏れがコンパイルを通ってしまう。手動プレイで初めて発覚しがち。

- **新しい `InteractionKind`**: 以下**すべて**に追加する。1つでも漏れると発動しない/表示されない。
  - `components/interactable.go` の `Config()`（発動方式 SameTile/Manual 等）
  - `activity/execute_interaction.go`（種別 → イベント発行）
  - `states/action_handlers.go` の `getInteractionActions`（発動可能アクション一覧。ここが漏れると Enter が無反応）
  - `activity/player_actions.go` の足元ログ（ここが漏れるとログも出ない）
  - 追加後、既存種別で照合する: `grep -rn "InteractionPortalNext" internal/ --include="*.go"` の出現箇所と件数に対し、新種別が同じ箇所・同数あることを確認する。
- **ステージ跨ぎクエリ（Phase 7 共存方式）**: 退避中ステージのエンティティを含みうるクエリは生の `ecs.NewFilterN` でなく `query.ActiveFilter` を使う。`Suspended` 除外を1箇所に集約している。漏れは `grep -rn "ecs.NewFilter" internal/ | grep -iE "GridElement|Door|SoloAI|Interactable"` 等で洗う。座標検索・破壊的操作（一括削除/平行移動）の漏れが特に致命的。
- 一般手順: 新種別/概念を足したら、**既存の1つを grep して全出現箇所を列挙し、同じ場所すべてに新種別を足したか照合**する。キーワード列挙の grep には盲点があるので、複数の同義キーワードで洗う。

## 参照

- 設計・コード規約の詳細は CLAUDE.md。
- コンポーネント追加の手順は add-component skill。
