---
name: ruins-tsp
description: tsp(TypeSpec)でOpenAPIスキーマを定義し Go/TS の型を生成するときの規約と手順。モデルや保存ファイルの型を追加・変更する、整数の幅を選ぶ、生成物を検証するときに使う。
---

# ruins tsp スキーマ規約

OpenAPI スキーマは `oas/typespec/` の tsp を単一ソースにし、Go と editor-ui の型を生成する。

## 生成パイプライン

- tsp は `oas/typespec/`。生成は `cd oas && make genall`。内訳は `make compile`(tsp→openapi.yml)・`make genapi`(→`internal/oapi`)・`make genfront`(→`editor-ui/src/generated`)・`make gendoc`。
- 生成物 `internal/oapi/*.gen.go` と `editor-ui/src/generated/*.ts` は手編集しない。tsp を直して再生成する。
- 新しい tsp ファイルは `main.tsp` に `import` を足す。
- 初回は `make toolinstall`(typespec の npm install)が要る。genfront は docker を使う。

## namespace とモデル

- HTTP operation を持たない保存ファイルやレポートは、models-only の sub-namespace にする。`save.tsp`(`RuinsEditorApi.SaveData`)・`balance.tsp`(`RuinsEditorApi.Balance`)が例。operation が無くてもモデルは Go/TS 両方へ生成される。
- sub-namespace のモデルは生成型名にプレフィックスが付く。`RuinsEditorApi.Balance` → `oapi.Balance*`、`SaveData` → `SaveData*`。関心の分離と名前衝突回避になる。
- 別 namespace の型は完全修飾名で参照する。例 `RuinsEditorApi.ColorChannel`。

## フィールド定義

- フィールドはすべて名前付きスカラーで定義し、モデルは参照だけを持つ。`types.tsp` の規約に揃える。doc コメントと制約(`@minValue`/`@maxValue`/pattern など)はスカラーに集約する。
- スカラーは Go/TS とも型エイリアスに生成され透過的。`scalar HP extends integer` → Go `type BalanceHP = int` / TS `type BalanceHP = number`。意味づけや制約を足しても構築・利用コードは変わらない。

## 整数の幅。int32 か integer か

判断基準は「消費者」と「ドメインの表現」。

- 言語非依存の外部消費者がいて範囲の契約が要る → `int32`/`int64`。
- 消費者が Go と TS だけで、ドメインを Go の `int` で計算する → `integer`(→ Go `int`)。`int32` にするとドメインの `int` から `int32(...)` キャストが要る。TS はどちらも `number` で差が無い。
- 既存スキーマが `int32` で、Go が生成 `int32` 型を直接使いキャストが無いなら、そのままでよい。整合のためだけに広く置換すると編集API契約・serde・呼び出し側へ波及し、費用対効果が悪い。

## null と optional

- required 配列は producer が空スライスで初期化し `[]` を出す。nil スライスは JSON `null` になり、消費側の `map`/反復が落ちる。
- optional フィールドは Go ポインタ + `omitempty` になる。`append` する配列は required にしてポインタ地獄を避ける。単発の任意オブジェクトだけ optional にする。

## 検証

- 生成物はスキーマで検証する。`raw.ValidateRaws` に倣い、`oapi.GetSpec()` → `Components.Schemas["<Namespace>.<Model>"]` → `VisitJSON`。コンポーネント名はドット付き。例 `"Balance.Report"`。
- 適合(正)と違反(負)の両方向をテストする。正だけでは制約が効いている保証にならない。違反 JSON を1箇所作り `VisitJSON` がエラーを返すことを確認する。

## 変更後の検証

- `cd oas && make genall` で再生成する。
- Go は `make check`。editor-ui は `cd editor-ui && npm run build`(tsc+vite) と `npm run lint`。
- 生成物の差分もコミットする。
