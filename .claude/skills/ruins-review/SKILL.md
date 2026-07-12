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

## 参照

- 設計・コード規約の詳細は CLAUDE.md。
- コンポーネント追加の手順は add-component skill。
