package mapplanner

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
)

// ConvertIsolatedWalls は床に隣接しない壁を指定したタイルに変換するプランナー
// オートタイル計算前に実行することで、正しい接続パターンで壁タイルを生成する
type ConvertIsolatedWalls struct {
	// ReplacementTile は孤立した壁を置き換えるタイル名
	ReplacementTile string
}

// NewConvertIsolatedWalls は新しいConvertIsolatedWallsプランナーを作成する
// replacementTile: 孤立した壁を置き換えるタイル名（例: "void", "floor"）
func NewConvertIsolatedWalls(replacementTile string) ConvertIsolatedWalls {
	return ConvertIsolatedWalls{
		ReplacementTile: replacementTile,
	}
}

// PlanMeta は床に隣接しない壁タイルを指定したタイルに変換する
func (c ConvertIsolatedWalls) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 変換が他のタイルの判定に影響しないように、まず変換すべきタイルを特定
	tilesToConvert := make([]int, 0)
	for i := range planData.Tiles {
		tile := planData.Tiles[i]

		// マップの端にあるタイルは変換対象外（境界として残す）
		x, y := planData.Level.XYTileCoord(resources.TileIdx(i))
		if int(x) == 0 || int(x) == width-1 || int(y) == 0 || int(y) == height-1 {
			continue
		}

		// wallタイルで床に隣接していない場合、変換対象とする
		if tile.Name == consts.TileNameWall && !planData.AdjacentAnyFloor(resources.TileIdx(i)) {
			tilesToConvert = append(tilesToConvert, i)
		}
	}

	// 特定したタイルを変換
	for _, i := range tilesToConvert {
		planData.Tiles[i] = planData.GetTile(c.ReplacementTile)
	}

	return nil
}
