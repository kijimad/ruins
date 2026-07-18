package consts

// ChunkX はチャンク単位の東西量。1 ＝ チャンク1枚ぶん。
//
// タイル単位の Tile や絶対タイルの worldstream.AbsTileX とは別物で、チャンクの絶対インデックスや
// 帯のチャンク数といった「チャンクで数える量」を型で区別する。東西1次元の帯なので X のみで足りる。
// タイルへは Tiles() で明示的に変換する。
type ChunkX int

// Tiles はチャンク量をタイル幅へ変換する。cx チャンク ＝ cx × chunkW タイル。
func (cx ChunkX) Tiles(chunkW Tile) Tile {
	return Tile(int(cx)) * chunkW
}
