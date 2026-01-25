package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
)

// RoomDraw は部屋を描画するビルダー
type RoomDraw struct{}

// PlanMeta はメタデータをビルドする
func (b RoomDraw) PlanMeta(planData *MetaPlan) error {
	b.build(planData)
	return nil
}

func (b RoomDraw) build(planData *MetaPlan) {
	for _, room := range planData.Rooms {
		b.rectangle(planData, room)
	}
}

func (b RoomDraw) rectangle(planData *MetaPlan, room gc.Rect) {
	for x := room.X1; x <= room.X2; x++ {
		for y := room.Y1; y <= room.Y2; y++ {
			idx := planData.Level.XYTileIndex(x, y)
			if 0 < int(idx) && int(idx) < int(planData.Level.TileWidth)*int(planData.Level.TileHeight)-1 {
				planData.Tiles[idx] = planData.GetTile("floor")
			}
		}
	}
}
