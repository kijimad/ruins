package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// ConvertIsolatedWalls は床に隣接しない壁を指定したタイルに変換するプランナー
// オートタイル計算前に実行することで、正しい接続パターンで壁タイルを生成する
type ConvertIsolatedWalls struct {
	// ReplacementTile は孤立した壁を置き換えるタイル名
	ReplacementTile string
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
		pos := planData.Level.IndexToCoord(gc.TileIdx(i))
		if int(pos.X) == 0 || int(pos.X) == width-1 || int(pos.Y) == 0 || int(pos.Y) == height-1 {
			continue
		}

		// wallタイルで床に隣接していない場合、変換対象とする
		if tile.Name == consts.TileNameWall && !planData.AdjacentAnyFloor(gc.TileIdx(i)) {
			tilesToConvert = append(tilesToConvert, i)
		}
	}

	// 特定したタイルを変換
	for _, i := range tilesToConvert {
		planData.Tiles[i] = planData.GetTile(c.ReplacementTile)
	}

	return nil
}
