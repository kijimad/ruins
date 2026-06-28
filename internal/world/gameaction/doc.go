// Package gameaction は query と lifecycle を組み合わせた複合的なゲームロジックを提供する。
//
// query パッケージと lifecycle パッケージの両方に依存する。
//
// # 分類ルール
//
// このパッケージに配置する関数は、複数の query/lifecycle 操作を組み合わせて
// ゲームルールを実現する複合操作である。
//
// 例: ダメージ処理（HP変更 + Dead付与 + ログ出力）、購入処理（通貨消費 + アイテム生成）
package gameaction
