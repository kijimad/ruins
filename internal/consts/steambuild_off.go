//go:build !steam

package consts

// IsSteamBuild は steam タグ無しビルドでは false。既定配布と WASM が該当し体験版になる。
const IsSteamBuild = false
