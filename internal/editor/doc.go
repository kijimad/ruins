// Package editor はゲームデータ（raw.toml）を編集するWebエディタを提供する。
//
// ローカルHTTPサーバーを起動し、ブラウザ上でアイテム等のマスターデータを
// 閲覧・編集できる。標準HTMLフォームによるPRGパターンで、フォーム送信のたびに
// raw.tomlファイルを直接書き換える。
//
// 使い方:
//
//	go run . editor
//
// 責務:
//   - raw.tomlの読み込み・書き出し
//   - HTTPハンドラの提供
//   - HTMLテンプレートのレンダリング
package editor
