---
name: ruins-cue
description: raw のスキーマを CUE で書き換えるときの規約と手順。raw.toml に型・フィールド・列挙値・参照・制約を足すとき、internal/rawschema の CUE や生成器を触るときに使う
---

# ruins CUE スキーマ規約

raw のデータの形・値の制約・行の間の参照は `internal/rawschema/*.cue` を唯一の源泉にする。

## 配置

- `schema.cue`: 型と制約。定義 `#X` が Go の型になる。
- `refs.cue`: 参照を引く id の索引。隠しフィールドに置く。
- `root.cue`: 検証するデータの最上位を `#Raws` に結ぶ。
- `gen/`: CUE から `models_gen.go` を書き出す生成器。`go generate` から呼ぶ。
- `rawcheck/`: CUE で raw を検証する。テストからだけ使い、ゲームのバイナリには CUE を入れない。

## 変更の手順

1. `schema.cue` を直す。
2. `make generate` で `models_gen.go` を再生成する。生成物は手編集しない。
3. `make fmt` で CUE を整形する。中身は `go tool cue fmt ./internal/rawschema/`。
4. `make check` で検証する。生成物の差分もコミットする。

## 型の書き方

- スカラーは名前付きの定義にし、型の土台を必ず明示する。`#EntryWeight: number & >=0` のように書く。`>=0` だけでは Go の型が決まらない。
- 整数は既定で `int` にし Go の `int` に写す。幅を固定したいときだけ `@rawgen(type=int32)` を付ける。ドメインを Go の `int` で計算するなら `int` のままにし、キャストを生まない。
- 列挙は値の選言で書く。`"a" | "b"` は Go の `type X string` と定数と `Valid()` と `XValues()` になる。
  - 定数名は値をキャメルケースにしたもの。数字で始まる値には `N` を前置する。
  - 他の型の定数と名前がぶつかるときは `@rawgen(prefix)` で型名を前置する。
  - 数の列挙は `@rawgen(enum=float32)` のように土台の型を指定する。
- フィールドは `name!:` が必須、`name?:` が任意。任意は Go でポインタと `omitempty` になる。
  - `append` する配列は必須にしてポインタを避ける。単発の任意オブジェクトだけ任意にする。
- 定義は閉じている。未知のフィールドは検証で落ちる。

## 参照と重複

- 参照先の表ごとに `refs.cue` へ索引を置く。`_itemIDs: {if items != _|_ for x in items {(x.id): true}}`。表を省いたデータでも索引が空になるよう `if items != _|_` で囲み、`items?: _` を最上位に宣言する。
- 参照する側は通常フィールド `<field>_ref` で索引を引き、`@rawgen(-)` で Go から外す。無い id は undefined field で落ちる。
  - 任意フィールドは `if f != _|_ { f_ref: _xIDs[f] @rawgen(-) }`。
  - 配列は `parts_ref: [for x in parts {_bodyPartIDs[x]}] @rawgen(-)`。
- 検査は必ず通常フィールドに置く。隠しフィールド `_x` の失敗は検証で報告されない。
- 選択肢は `or()` の選言でなく索引の参照で引く。選言は実データで桁違いに遅い。
- 既定値の `*[] | _` は Go の `Unify` で落ちるので使わない。省略の判定は `if x != _|_` で書く。
- 表が自分自身を参照する索引は、表の定義の中に置くと評価が循環する。`#Raws` の通常フィールドに置く。`tileSpawns_ref` が例。
- id の重複は `#Raws` に `<x>Ids_unique: {if xs != _|_ for i, x in xs {(x.id): i}} @rawgen(-)` を置く。同じ id に別の添字が入り、値の衝突で落ちる。
- 複数の表の集計や件数など CUE で素直に書けない規則は Go の `raw.validateRules` に置く。

## 検証とテスト

- `rawcheck.Validate` は Go へ読み込んだ `Raws` を JSON にしてから検証する。TOML で省いた必須フィールドも Go のゼロ値として現れる。
- 違反は `*rawcheck.Error` で返る。`errors.As` で取り出し、`Paths()` で違反したパスを `assert.Equal` で固定する。文言では判定しない。
- 正と負の両方向をテストする。実データが通ることと、1か所壊したデータがそのパスで落ちることを確かめる。
- 参照と重複の規則を足したら `rawcheck/refs_test.go` の表に行を足す。
