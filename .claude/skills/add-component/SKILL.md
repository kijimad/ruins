---
name: add-component
description: ruins に新しい ECS コンポーネントを追加する手順。型定義・登録表・生成・serde除外・クエリ設計・テストの漏れを防ぐ。コンポーネントやマーカーを増やすときに使う
---

# コンポーネント追加手順

ruins のコンポーネントは登録表からコード生成される。手順を漏らすと保存破損や生成漏れになる。

## 手順

1. **型を定義する**
   - `internal/components/` に `type X struct{...}`（マーカーなら `struct{}`）。
   - doc コメントを付ける（体言止めは使わない）。

2. **登録表に1行追加する**
   - `internal/components/genspec/registry.go` の `Registry` に `{Field: "X", Comment: "…を保持する"}` を追加。
   - フィールド名＝型名にする（`Type` フィールドは廃止済み）。

3. **生成する**
   - `make generate` を実行。`internal/components/components_gen.go`（EntitySpec / Components / InitializeComponents / AddEntity）が再生成される。
   - `components_gen.go` は生成物なので手編集しない。

4. **serde（保存）の要否を判断する**
   - 丸ごと保存なので既定で全コンポーネントが保存される。
   - 一時・実行時のみの状態は `internal/save/serde.go` の `skipComponents()` に追加する。対象例: ダーティフラグ、キャッシュ、イベント、`sync.Mutex` を含む型、interface スライス、struct-keyed map、毎フレーム算出値。
   - 保存する場合は serde 安全性を守る: interface を持たない、map のキーは string にする（struct キーの map は `json:"-"` + ロード時再構築）。

5. **クエリ設計を選ぶ**
   - 「全◯◯」を archetype クエリする排他カテゴリは、Kind enum フィールドではなく**別マーカーコンポーネント**にする（Faction / Location が実例）。排他は lifecycle の関数で保証する。
   - 個別に読むだけならフィールドで十分。

6. **テストを追加する**
   - `TestAddEntity_AllFields` が EntitySpec↔AddEntity の網羅を自動検証する。
   - 挙動テストを追加する。

## 検証

- `make generate` に差分が出ないこと（CI の generate-check が担保）。
- `make check` が緑。
- 保存するコンポーネントなら save の往復テストが通ること。
