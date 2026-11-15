package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// RectRoomPlanner は長方形の部屋を作成する
type RectRoomPlanner struct {
	// Rooms は上端と下端の部屋のほかに置く部屋の数
	Rooms rawschema.RandInt
	// RoomSize は部屋の幅と高さ
	RoomSize rawschema.RandInt
}

// PlanInitial は初期プランを行う
func (b RectRoomPlanner) PlanInitial(planData *MetaPlan) error {
	b.PlanRooms(planData)
	return nil
}

// PlanRooms は部屋をプランする
// 上端(y=0)と下端(y=height)に必ず各1つの部屋を配置し、上下端への接続を保証する
func (b RectRoomPlanner) PlanRooms(planData *MetaPlan) {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)
	rooms := make([]gc.Rect, 0, 16)

	// 上端に必ず1つの部屋を配置（y=0から開始）
	topRoom := b.createRoom(planData, width, height, 0)
	rooms = append(rooms, topRoom)

	// 下端に必ず1つの部屋を配置（Y2=height-1に密着）
	bottomW := b.RoomSize.Roll(planData.RNG)
	bottomH := b.RoomSize.Roll(planData.RNG)
	bottomX := planData.RNG.IntN(width)
	rooms = append(rooms, gc.Rect{Min: consts.Coord[consts.Tile]{X: consts.Tile(bottomX), Y: consts.Tile(max(height-bottomH, 0))}, Max: consts.Coord[consts.Tile]{X: consts.Tile(min(bottomX+bottomW, width) - 1), Y: consts.Tile(height - 1)}})

	// 残りの部屋はランダム配置
	maxRooms := b.Rooms.Roll(planData.RNG)
	for range maxRooms {
		y := planData.RNG.IntN(height)
		rooms = append(rooms, b.createRoom(planData, width, height, y))
	}

	planData.Rooms = rooms
}

// createRoom は指定のY座標から部屋を生成する
func (b RectRoomPlanner) createRoom(planData *MetaPlan, width, height, y int) gc.Rect {
	x := planData.RNG.IntN(width)
	w := b.RoomSize.Roll(planData.RNG)
	h := b.RoomSize.Roll(planData.RNG)
	return gc.Rect{Min: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(max(y, 0))}, Max: consts.Coord[consts.Tile]{X: consts.Tile(min(x+w, width) - 1), Y: consts.Tile(min(y+h, height) - 1)}}

}
