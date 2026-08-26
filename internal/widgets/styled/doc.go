// Package styled は一覧描画の共通データ型と、枠付き背景の直接描画を提供する。
//
// # 責務
//
//   - 一覧の1行を表すセル型 Cell と、その組み立てヘルパ TextCell・IconCell・TextCells。
//     セルは文字列かアイコンのどちらかを持ち、揃え方向は TextAlign で表す。
//   - パネルやログ領域の枠付き背景を screen へ直接描く DrawFramedBackground と、その既定スタイル PanelStyle。
//
// 実際のウィジェット組み立ては internal/ui のツリーが担う。styled は描画の土台となる
// データ型とスタイルだけを持ち、状態やエンティティには関わらない。
//
// # 使い分け
//
//   - 一覧の行データを組みたい: Cell・TextCell・TextCells・IconCell・TextAlign を使う。
//   - パネルの背景枠を敷きたい: DrawFramedBackground と PanelStyle を使う。
package styled
