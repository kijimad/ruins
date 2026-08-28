// Package styled は一覧描画の共通データ型を提供する。
//
// # 責務
//
//   - 一覧の1行を表すセル型 Cell と、その組み立てヘルパ TextCell・IconCell・TextCells。
//     セルは文字列かアイコンのどちらかを持ち、揃え方向は TextAlign で表す。
//   - 一覧の列を役割で宣言する Col と、その構成ヘルパ Name・Num・Icon・Cols。
//     生のピクセル幅の代わりに「名前列・数値列・アイコン列」で列を組む。
//
// 実際のウィジェット組み立ては internal/ui のツリーが、メニューの枠組みは menuframe が担う。
// styled は描画の土台となるデータ型だけを持ち、状態やエンティティには関わらない。
//
// # 使い分け
//
//   - 一覧の行データを組みたい: Cell・TextCell・TextCells・IconCell を使う。
//   - 一覧の列を宣言したい: Cols・Name・Num・Icon を使う。
package styled
