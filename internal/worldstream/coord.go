package worldstream

import "github.com/kijimaD/ruins/internal/consts"

// BandOriginY は northIndex（北進したチャンク数）と chunkH から帯の絶対原点 Y を返す。
// 帯ローカル Y=0 すなわち北端が絶対軸で指す位置。北は -Y なので northIndex ぶん負へ伸びる。
func BandOriginY(northIndex consts.Chunk, chunkH consts.Tile) consts.AbsTileY {
	return consts.AbsTileY(-int(northIndex) * int(chunkH))
}

// ToAbsY は帯ローカル Y を絶対 Y に変換する。absY = localY + bandOriginY。
func ToAbsY(bandOriginY consts.AbsTileY, localY consts.Tile) consts.AbsTileY {
	return bandOriginY + consts.AbsTileY(localY)
}

// ToLocalY は絶対 Y を帯ローカル Y に変換する。localY = absY - bandOriginY。
func ToLocalY(bandOriginY consts.AbsTileY, absY consts.AbsTileY) consts.Tile {
	return consts.Tile(absY - bandOriginY)
}
