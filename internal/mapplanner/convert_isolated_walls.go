package mapplanner

import (
	"github.com/kijimaD/ruins/internal/resources"
)

// ConvertIsolatedWallsToFloor は床に隣接しない壁をfloorに変換するプランナー
// オートタイル計算前に実行することで、正しい接続パターンで壁タイルを生成する
type ConvertIsolatedWallsToFloor struct{}

// NewConvertIsolatedWallsToFloor は新しいConvertIsolatedWallsToFloorプランナーを作成する
func NewConvertIsolatedWallsToFloor() ConvertIsolatedWallsToFloor {
	return ConvertIsolatedWallsToFloor{}
}

// PlanMeta は床に隣接しない壁タイルをfloorタイルに変換する
func (c ConvertIsolatedWallsToFloor) PlanMeta(planData *MetaPlan) error {
	// 変換が他のタイルの判定に影響しないように、まず変換すべきタイルを特定
	tilesToConvert := make([]int, 0)
	for i := range planData.Tiles {
		tile := planData.Tiles[i]

		// wallタイルで床に隣接していない場合、変換対象とする
		if tile.Name == "wall" && !planData.AdjacentAnyFloor(resources.TileIdx(i)) {
			tilesToConvert = append(tilesToConvert, i)
		}
	}

	// 特定したタイルを変換
	for _, i := range tilesToConvert {
		planData.Tiles[i] = planData.GetTile("floor")
	}

	return nil
}
