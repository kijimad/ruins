package worldstream

import "github.com/kijimaD/ruins/internal/consts"

// AbsTileX は東西の絶対タイル X 座標。
//
// 東へ進むほど無限に増える絶対軸で、帯ローカルの GridElement.X（常に 0..K*chunkW の有界）
// とは別物。両者の取り違え（絶対と局所の混同）を Go の型で弾くための別名型。
// 前線 frontEast・帯原点 bandOriginX・到達距離スコアはこの絶対軸で扱う。
// 南北はストリーミングしない（高さ固定の帯）ため、絶対軸は X のみで足りる。
// 詳細設計は docs/design/20260717_60.md §9。
type AbsTileX int

// BandOriginX は eastIndex（東進したチャンク数）と chunkW から帯の絶対原点 X を返す。
// 帯ローカル X=0 が絶対軸で指す位置。
func BandOriginX(eastIndex int, chunkW consts.Tile) AbsTileX {
	return AbsTileX(eastIndex * int(chunkW))
}

// ToAbs は帯ローカル X を絶対 X に変換する（absX = localX + bandOriginX）。
func ToAbs(bandOriginX AbsTileX, localX consts.Tile) AbsTileX {
	return bandOriginX + AbsTileX(localX)
}

// ToLocal は絶対 X を帯ローカル X に変換する（localX = absX - bandOriginX）。
func ToLocal(bandOriginX AbsTileX, absX AbsTileX) consts.Tile {
	return consts.Tile(absX - bandOriginX)
}
