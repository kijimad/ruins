// Package steam はSteamworks SDKとの統合を提供する。
//
// ビルドタグ "steam" を指定した場合のみSteamworksの初期化が行われる。
// タグなしでビルドした場合はすべての関数がno-opになる。
//
// 使い分け:
//   - 開発時・WASM: タグなしでビルドする。Steam依存が発生しない
//   - Steamリリース: `-tags steam` でビルドする。Steamクライアント経由での起動が必要になる
//
// 責務:
//   - Steamworks APIの初期化と終了処理
//   - Steam経由での起動チェック（RestartAppIfNecessary）
package steam
