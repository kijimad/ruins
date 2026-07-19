package consts

// Chunk はチャンクで数える量。1 ＝ チャンク1枚ぶん。
//
// タイル単位の Tile や絶対タイルの AbsTileX とは別物で、チャンクの絶対インデックスや
// 帯のチャンク数といった「チャンクで数える量」を型で区別する。
// Tile と同じく軸に依らないスカラーで、東西インデックスにも枚数にも使う。X を焼き込まない。
// タイルへは Tiles() で明示的に変換する。
type Chunk int

// Tiles はチャンク量をタイル数へ変換する。c チャンク ＝ c × chunkSize タイル。
// chunkSize はその軸のチャンクの辺のタイル数。東西なら chunkW。
func (c Chunk) Tiles(chunkSize Tile) Tile {
	return Tile(int(c)) * chunkSize
}
