package worldstream

import "github.com/kijimaD/ruins/internal/consts"

// ToAbsY は帯ローカル Y を絶対 Y に変換する。absY = localY + bandOriginY。
func ToAbsY(bandOriginY consts.AbsTileY, localY consts.Tile) consts.AbsTileY {
	return bandOriginY + consts.AbsTileY(localY)
}

// ToLocalY は絶対 Y を帯ローカル Y に変換する。localY = absY - bandOriginY。
func ToLocalY(bandOriginY consts.AbsTileY, absY consts.AbsTileY) consts.Tile {
	return consts.Tile(absY - bandOriginY)
}
