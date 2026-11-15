package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// RuinsPlanner は廃墟風レイアウトを生成するビルダー
// 建物の残骸や瓦礫が散在する廃墟を作成
type RuinsPlanner struct {
	Layout rawschema.RuinsLayout
}

// PlanInitial は初期廃墟マップをビルドする
func (r RuinsPlanner) PlanInitial(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	ruinCount := r.Layout.Buildings.Roll(planData.RNG)

	for range ruinCount {
		// ランダムな位置とサイズで廃墟の基盤を作成
		buildingWidth := r.Layout.BuildingWidth.Roll(planData.RNG)
		buildingHeight := r.Layout.BuildingHeight.Roll(planData.RNG)

		// IntNの引数が正であることを保証
		maxXRange := max(1, width-buildingWidth-4)
		maxYRange := max(1, height-buildingHeight-4)
		x := 2 + planData.RNG.IntN(maxXRange)
		y := 2 + planData.RNG.IntN(maxYRange)

		// 建物の外壁を作成
		room := gc.Rect{Min: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}, Max: consts.Coord[consts.Tile]{X: consts.Tile(x + buildingWidth - 1), Y: consts.Tile(y + buildingHeight - 1)}}

		planData.Rooms = append(planData.Rooms, room)
	}
	return nil
}

// RuinsDraw は廃墟構造を描画する
type RuinsDraw struct {
	Layout rawschema.RuinsLayout
}

// PlanMeta は廃墟構造をタイルに描画する
func (r RuinsDraw) PlanMeta(planData *MetaPlan) error {
	// まず全体を床で埋める（屋外エリア）
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile(r.Layout.Tiles.Floor)
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
	for x := building.Min.X; x <= building.Max.X; x++ {
		// 上辺
		if y := building.Min.Y; planData.RNG.Float64() > r.Layout.WallGapChance {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
			planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall)
		}
		// 下辺
		if y := building.Max.Y; planData.RNG.Float64() > r.Layout.WallGapChance {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
			planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall)
		}
	}

	for y := building.Min.Y; y <= building.Max.Y; y++ {
		// 左辺
		if x := building.Min.X; planData.RNG.Float64() > r.Layout.WallGapChance {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
			planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall)
		}
		// 右辺
		if x := building.Max.X; planData.RNG.Float64() > r.Layout.WallGapChance {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
			planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall)
		}
	}

	// 建物内部に部屋の仕切りを作成（一部破損）
	r.addInteriorWalls(planData, building)
}

// addInteriorWalls は建物内部に仕切り壁を追加する
func (r RuinsDraw) addInteriorWalls(planData *MetaPlan, building gc.Rect) {
	buildingWidth := int(building.Width())
	buildingHeight := int(building.Height())

	// 建物が十分大きい場合のみ内部の仕切りを作成
	if buildingWidth >= 10 && buildingHeight >= 8 {
		// 縦の仕切り
		midX := building.Min.X + consts.Tile(buildingWidth/2)
		for y := building.Min.Y + 2; y <= building.Max.Y-2; y++ {
			if planData.RNG.Float64() > r.Layout.PartitionGapChance {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: midX, Y: y})
				planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall)
			}
		}

		// 横の仕切り
		midY := building.Min.Y + consts.Tile(buildingHeight/2)
		for x := building.Min.X + 2; x <= building.Max.X-2; x++ {
			if planData.RNG.Float64() > r.Layout.PartitionGapChance {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: midY})
				planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall)
			}
		}
	}
}

// RuinsDebris は瓦礫や破片を配置する
type RuinsDebris struct {
	Layout rawschema.RuinsLayout
}

// PlanMeta は廃墟に瓦礫を配置する
func (r RuinsDebris) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 屋外エリアに瓦礫を散乱させる
	for x := 1; x < width-1; x++ {
		for y := 1; y < height-1; y++ {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})

			if planData.Tiles[idx].Id == r.Layout.Tiles.Floor {
				// 建物から離れた場所ほど瓦礫が少ない
				debrisChance := r.calculateDebrisChance(planData, x, y)

				if planData.RNG.Float64() < debrisChance {
					planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Wall) // 瓦礫として壁タイルを使用
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
		centerX := float64(room.Min.X+room.Max.X) / 2.0
		centerY := float64(room.Min.Y+room.Max.Y) / 2.0

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
	Layout rawschema.RuinsLayout
}

// PlanMeta は廃墟間に通路を作成する
func (r RuinsCorridors) PlanMeta(planData *MetaPlan) error {
	if len(planData.Rooms) < 2 {
		return nil
	}

	// 各建物間に通路を作成
	for i := range len(planData.Rooms) {
		for j := i + 1; j < len(planData.Rooms); j++ {
			room1 := planData.Rooms[i]
			room2 := planData.Rooms[j]

			if planData.RNG.Float64() < r.Layout.PathChance {
				r.createRuinedPath(planData, room1, room2)
			}
		}
	}
	return nil
}

// createRuinedPath は破損した通路を作成する
func (r RuinsCorridors) createRuinedPath(planData *MetaPlan, room1, room2 gc.Rect) {
	// 各建物の中心を計算
	center1X := (room1.Min.X + room1.Max.X) / 2
	center1Y := (room1.Min.Y + room1.Max.Y) / 2
	center2X := (room2.Min.X + room2.Max.X) / 2
	center2Y := (room2.Min.Y + room2.Max.Y) / 2

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
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: currentX, Y: currentY})
			if planData.Tiles[idx].Id == r.Layout.Tiles.Wall {
				planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Floor)
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
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: currentX, Y: currentY})
			if planData.Tiles[idx].Id == r.Layout.Tiles.Wall {
				planData.Tiles[idx] = planData.GetTile(r.Layout.Tiles.Floor)
			}
		}
	}
}

// NewRuinsPlanner は間取りの行 l の廃墟ビルダーを作成する
func NewRuinsPlanner(l rawschema.RuinsLayout, width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(RuinsPlanner{Layout: l})
	chain.With(NewFillAll(l.Tiles.Wall))
	chain.With(RuinsDraw{Layout: l})
	chain.With(RuinsDebris{Layout: l})
	chain.With(RuinsCorridors{Layout: l})
	chain.With(ConvertIsolatedWalls{WallTile: l.Tiles.Wall, ReplacementTile: l.Tiles.Void})
	chain.With(EnvironmentPlanner{})

	return chain, nil
}
