package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
)

// RectRoomPlanner は長方形の部屋を作成する
type RectRoomPlanner struct{}

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
	rooms := []gc.Rect{}

	// 上端に必ず1つの部屋を配置（y=0から開始）
	rooms = append(rooms, b.createRoom(planData, width, height, 0))

	// 下端に必ず1つの部屋を配置（Y2=heightに密着）
	bottomW := 2 + planData.RNG.IntN(8)
	bottomH := 2 + planData.RNG.IntN(8)
	bottomX := planData.RNG.IntN(width)
	rooms = append(rooms, gc.Rect{
		X1: gc.Tile(bottomX),
		X2: gc.Tile(min(bottomX+bottomW, width)),
		Y1: gc.Tile(max(height-bottomH, 0)),
		Y2: gc.Tile(height),
	})

	// 残りの部屋はランダム配置
	maxRooms := 4 + planData.RNG.IntN(10)
	for i := 0; i < maxRooms; i++ {
		y := planData.RNG.IntN(height)
		rooms = append(rooms, b.createRoom(planData, width, height, y))
	}

	planData.Rooms = rooms
}

// createRoom は指定のY座標から部屋を生成する
func (b RectRoomPlanner) createRoom(planData *MetaPlan, width, height, y int) gc.Rect {
	x := planData.RNG.IntN(width)
	w := 2 + planData.RNG.IntN(8)
	h := 2 + planData.RNG.IntN(8)
	return gc.Rect{
		X1: gc.Tile(x),
		X2: gc.Tile(min(x+w, width)),
		Y1: gc.Tile(max(y, 0)),
		Y2: gc.Tile(min(y+h, height)),
	}
}
