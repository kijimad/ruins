package worldstream

import "github.com/kijimaD/ruins/internal/consts"

// 絶対タイル X の型 consts.AbsTileX は consts に移した。ここでは帯ローカルとの変換だけ提供する。

// BandOriginX は eastIndex（東進したチャンク数）と chunkW から帯の絶対原点 X を返す。
// 帯ローカル X=0 が絶対軸で指す位置。
func BandOriginX(eastIndex consts.Chunk, chunkW consts.Tile) consts.AbsTileX {
	return consts.AbsTileX(int(eastIndex) * int(chunkW))
}

// ToAbs は帯ローカル X を絶対 X に変換する。absX = localX + bandOriginX。
func ToAbs(bandOriginX consts.AbsTileX, localX consts.Tile) consts.AbsTileX {
	return bandOriginX + consts.AbsTileX(localX)
}

// ToLocal は絶対 X を帯ローカル X に変換する。localX = absX - bandOriginX。
func ToLocal(bandOriginX consts.AbsTileX, absX consts.AbsTileX) consts.Tile {
	return consts.Tile(absX - bandOriginX)
}
