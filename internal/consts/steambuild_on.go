//go:build steam

package consts

// IsSteamBuild は steam タグ付きでビルドされたかどうか。build_steam.sh の -tags steam で true になる。
// Steam 版は本番のフル版で、セーブ・ロードを有効にする。
const IsSteamBuild = true
