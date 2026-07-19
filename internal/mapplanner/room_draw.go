package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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
	for x := room.Min.X; x <= room.Max.X; x++ {
		for y := room.Min.Y; y <= room.Max.Y; y++ {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
			if 0 < int(idx) && int(idx) < int(planData.Level.TileWidth)*int(planData.Level.TileHeight)-1 {
				planData.Tiles[idx] = planData.GetTile("floor")
			}
		}
	}
}
