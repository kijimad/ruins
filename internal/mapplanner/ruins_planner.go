package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
)

// RuinsPlanner は廃墟風レイアウトを生成するビルダー
// 建物の残骸や瓦礫が散在する廃墟を作成
type RuinsPlanner struct{}

// PlanInitial は初期廃墟マップをビルドする
func (r RuinsPlanner) PlanInitial(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// まず大きな廃墟建物の基盤を作成（3-5個の大きな矩形）
	ruinCount := 3 + planData.RNG.IntN(3)

	for i := 0; i < ruinCount; i++ {
		// ランダムな位置とサイズで廃墟の基盤を作成
		buildingWidth := 8 + planData.RNG.IntN(10)
		buildingHeight := 6 + planData.RNG.IntN(8)

		// IntNの引数が正であることを保証
		maxXRange := max(1, width-buildingWidth-4)
		maxYRange := max(1, height-buildingHeight-4)
		x := 2 + planData.RNG.IntN(maxXRange)
		y := 2 + planData.RNG.IntN(maxYRange)

		// 建物の外壁を作成
		room := gc.Rect{
			X1: gc.Tile(x),
			Y1: gc.Tile(y),
			X2: gc.Tile(x + buildingWidth - 1),
			Y2: gc.Tile(y + buildingHeight - 1),
		}
		planData.Rooms = append(planData.Rooms, room)
	}
	return nil
}

// RuinsDraw は廃墟構造を描画する
type RuinsDraw struct {
	FloorTile string
	WallTile  string
}

// PlanMeta は廃墟構造をタイルに描画する
func (r RuinsDraw) PlanMeta(planData *MetaPlan) error {
	// まず全体を床で埋める（屋外エリア）
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile(r.FloorTile)
	}

	// 各廃墟建物を処理
	for _, room := range planData.Rooms {
		r.drawRuinedBuilding(planData, room)
	}
	return nil
}

// drawRuinedBuilding は破損した建物を描画する
func (r RuinsDraw) drawRuinedBuilding(planData *MetaPlan, building gc.Rect) {
	// 建物の外壁を描画（一部欠損あり）
	for x := building.X1; x <= building.X2; x++ {
		// 上辺
		if y := building.Y1; planData.RNG.Float64() > 0.3 { // 70%の確率で壁
			idx := planData.Level.XYTileIndex(x, y)
			planData.Tiles[idx] = planData.GetTile(r.WallTile)
		}
		// 下辺
		if y := building.Y2; planData.RNG.Float64() > 0.3 {
			idx := planData.Level.XYTileIndex(x, y)
			planData.Tiles[idx] = planData.GetTile(r.WallTile)
		}
	}

	for y := building.Y1; y <= building.Y2; y++ {
		// 左辺
		if x := building.X1; planData.RNG.Float64() > 0.3 {
			idx := planData.Level.XYTileIndex(x, y)
			planData.Tiles[idx] = planData.GetTile(r.WallTile)
		}
		// 右辺
		if x := building.X2; planData.RNG.Float64() > 0.3 {
			idx := planData.Level.XYTileIndex(x, y)
			planData.Tiles[idx] = planData.GetTile(r.WallTile)
		}
	}

	// 建物内部に部屋の仕切りを作成（一部破損）
	r.addInteriorWalls(planData, building)
}

// addInteriorWalls は建物内部に仕切り壁を追加する
func (r RuinsDraw) addInteriorWalls(planData *MetaPlan, building gc.Rect) {
	buildingWidth := int(building.X2 - building.X1 + 1)
	buildingHeight := int(building.Y2 - building.Y1 + 1)

	// 建物が十分大きい場合のみ内部の仕切りを作成
	if buildingWidth >= 10 && buildingHeight >= 8 {
		// 縦の仕切り
		midX := building.X1 + gc.Tile(buildingWidth/2)
		for y := building.Y1 + 2; y <= building.Y2-2; y++ {
			if planData.RNG.Float64() > 0.4 { // 60%の確率で壁
				idx := planData.Level.XYTileIndex(midX, y)
				planData.Tiles[idx] = planData.GetTile(r.WallTile)
			}
		}

		// 横の仕切り
		midY := building.Y1 + gc.Tile(buildingHeight/2)
		for x := building.X1 + 2; x <= building.X2-2; x++ {
			if planData.RNG.Float64() > 0.4 { // 60%の確率で壁
				idx := planData.Level.XYTileIndex(x, midY)
				planData.Tiles[idx] = planData.GetTile(r.WallTile)
			}
		}
	}
}

// RuinsDebris は瓦礫や破片を配置する
type RuinsDebris struct {
	FloorTile string
	WallTile  string
}

// PlanMeta は廃墟に瓦礫を配置する
func (r RuinsDebris) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 屋外エリアに瓦礫を散乱させる
	for x := 1; x < width-1; x++ {
		for y := 1; y < height-1; y++ {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

			if planData.Tiles[idx].Name == r.FloorTile {
				// 建物から離れた場所ほど瓦礫が少ない
				debrisChance := r.calculateDebrisChance(planData, x, y)

				if planData.RNG.Float64() < debrisChance {
					planData.Tiles[idx] = planData.GetTile(r.WallTile) // 瓦礫として壁タイルを使用
				}
			}
		}
	}
	return nil
}

// calculateDebrisChance は瓦礫の配置確率を計算する
func (r RuinsDebris) calculateDebrisChance(planData *MetaPlan, x, y int) float64 {
	// 最寄りの建物までの距離を計算
	minDistance := 1000.0

	for _, room := range planData.Rooms {
		// 建物の中心からの距離
		centerX := float64(room.X1+room.X2) / 2.0
		centerY := float64(room.Y1+room.Y2) / 2.0

		dx := float64(x) - centerX
		dy := float64(y) - centerY
		distance := dx*dx + dy*dy // 平方根は不要

		if distance < minDistance {
			minDistance = distance
		}
	}

	// 距離に基づいて瓦礫確率を計算（近いほど高確率）
	if minDistance < 25 { // 建物の近く
		return 0.15
	} else if minDistance < 100 { // 中距離
		return 0.08
	}
	// 遠距離
	return 0.03
}

// RuinsCorridors は廃墟間を繋ぐ通路を作成する
type RuinsCorridors struct {
	FloorTile string
	WallTile  string
}

// PlanMeta は廃墟間に通路を作成する
func (r RuinsCorridors) PlanMeta(planData *MetaPlan) error {
	if len(planData.Rooms) < 2 {
		return nil
	}

	// 各建物間に通路を作成
	for i := 0; i < len(planData.Rooms); i++ {
		for j := i + 1; j < len(planData.Rooms); j++ {
			room1 := planData.Rooms[i]
			room2 := planData.Rooms[j]

			// 30%の確率で通路を作成（すべての建物を繋ぐわけではない）
			if planData.RNG.Float64() < 0.3 {
				r.createRuinedPath(planData, room1, room2)
			}
		}
	}
	return nil
}

// createRuinedPath は破損した通路を作成する
func (r RuinsCorridors) createRuinedPath(planData *MetaPlan, room1, room2 gc.Rect) {
	// 各建物の中心を計算
	center1X := (room1.X1 + room1.X2) / 2
	center1Y := (room1.Y1 + room1.Y2) / 2
	center2X := (room2.X1 + room2.X2) / 2
	center2Y := (room2.Y1 + room2.Y2) / 2

	// L字型の通路を作成（一部破損）
	currentX, currentY := center1X, center1Y

	// 水平方向に移動
	for currentX != center2X {
		if currentX < center2X {
			currentX++
		} else {
			currentX--
		}

		// 70%の確率で通路を作成（部分的に破損）
		if planData.RNG.Float64() > 0.3 {
			idx := planData.Level.XYTileIndex(currentX, currentY)
			if planData.Tiles[idx].Name == r.WallTile {
				planData.Tiles[idx] = planData.GetTile(r.FloorTile)
			}
		}
	}

	// 垂直方向に移動
	for currentY != center2Y {
		if currentY < center2Y {
			currentY++
		} else {
			currentY--
		}

		// 70%の確率で通路を作成
		if planData.RNG.Float64() > 0.3 {
			idx := planData.Level.XYTileIndex(currentX, currentY)
			if planData.Tiles[idx].Name == r.WallTile {
				planData.Tiles[idx] = planData.GetTile(r.FloorTile)
			}
		}
	}
}

// NewRuinsPlanner は廃墟ビルダーを作成する
func NewRuinsPlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(RuinsPlanner{})
	chain.With(NewFillAll("Wall")) // 全体を壁で埋める
	chain.With(RuinsDraw{
		FloorTile: "Floor",
		WallTile:  "Wall",
	}) // 廃墟構造を描画
	chain.With(RuinsDebris{
		FloorTile: "Floor",
		WallTile:  "Wall",
	}) // 瓦礫を配置
	chain.With(RuinsCorridors{
		FloorTile: "Floor",
		WallTile:  "Wall",
	}) // 通路を作成
	chain.With(NewBoundaryWall("Wall")) // 最外周を壁で囲む

	return chain, nil
}
