package consts

// Chunk はチャンクで数える量。1 ＝ チャンク1枚ぶん。
//
// タイルの Tile や絶対タイルの AbsTileY とは別物。軸に依らないスカラーで、東西インデックスにも
// 枚数にも使い、X を焼き込まない。タイルへは Tiles() で明示的に変換する。
type Chunk int

// UrbanMinSpan は市街地の一辺の最小チャンク数。市街地の規模抽選と、各地区に基本施設を課す raw の検証が共有する。
const UrbanMinSpan Chunk = 2

// Tiles はチャンク量をタイル数へ変換する。c チャンク ＝ c × chunkSize タイル。
// chunkSize はその軸のチャンクの辺のタイル数。東西なら chunkW。
func (c Chunk) Tiles(chunkSize Tile) Tile {
	return Tile(int(c)) * chunkSize
}
