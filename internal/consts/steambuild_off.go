//go:build !steam

package consts

// IsSteamBuild は steam タグ付きでビルドされたかどうか。タグなしのビルドでは false になる。
// 既定配布と WASM はこちらで、体験版として扱う。
const IsSteamBuild = false
