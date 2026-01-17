package mapplanner

import (
	"github.com/kijimaD/ruins/internal/resources"
)

// BoundaryWall は外枠をすべて指定タイルで覆うビルダー
// マップの最外枠のタイルを無条件で指定タイルタイプに変更する
type BoundaryWall struct {
	// WallTileName 外枠に使用するタイル名
	WallTileName string
}

// NewBoundaryWall は新しいBoundaryWallビルダーを作成する
func NewBoundaryWall(wallTileName string) BoundaryWall {
	return BoundaryWall{
		WallTileName: wallTileName,
	}
}

// PlanMeta はメタデータをビルドする
func (b BoundaryWall) PlanMeta(planData *MetaPlan) error {
	// 全タイルをチェックして最外枠のタイルを壁で覆う
	for i := range planData.Tiles {
		idx := resources.TileIdx(i)

		// 最外枠タイルの場合は無条件で壁にする
		if b.isBoundaryTile(planData, idx) {
			planData.Tiles[idx] = planData.GetTile(b.WallTileName)
		}
	}
	return nil
}

// isBoundaryTile はマップの最外枠のタイルかを判定する
func (b BoundaryWall) isBoundaryTile(planData *MetaPlan, idx resources.TileIdx) bool {
	x, y := planData.Level.XYTileCoord(idx)
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	return int(x) == 0 || int(x) == width-1 || int(y) == 0 || int(y) == height-1
}
