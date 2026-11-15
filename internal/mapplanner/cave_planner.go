package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// CavePlanner は洞窟風レイアウトを生成するビルダー
// セルラーオートマトンを使用して有機的な洞窟形状を作成
// TODO: 全然わかってないので理解する
type CavePlanner struct {
	Row   rawschema.NoiseStart
	Tiles rawschema.LayoutTiles
}

// PlanInitial は初期洞窟マップをプランする
func (c CavePlanner) PlanInitial(planData *MetaPlan) error {
	for i := range planData.Tiles {
		if planData.RNG.Float64() < c.Row.WallChance {
			planData.Tiles[i] = planData.GetTile(c.Tiles.Wall)
		} else {
			planData.Tiles[i] = planData.GetTile(c.Tiles.Floor)
		}
	}

	return nil
}

// CaveCellularAutomata はセルラーオートマトンによる洞窟生成
type CaveCellularAutomata struct {
	Row   rawschema.AutomataStep
	Tiles rawschema.LayoutTiles
}

// PlanMeta はセルラーオートマトンで洞窟を生成する
func (c CaveCellularAutomata) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// セルラーオートマトンを指定回数実行
	for range c.Row.Iterations {
		newTiles := make([]rawschema.Tile, len(planData.Tiles))

		for x := range width {
			for y := range height {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})

				// 左右端は壁にする
				// 上下端はオートマトンのルールに任せる
				if x == 0 || x == width-1 {
					newTiles[idx] = planData.GetTile(c.Tiles.Wall)
					continue
				}

				// 周囲の壁の数を数える
				wallCount := c.countWallsInRadius(planData, x, y, 1)

				if wallCount >= c.Row.WallNeighbors {
					newTiles[idx] = planData.GetTile(c.Tiles.Wall)
				} else {
					newTiles[idx] = planData.GetTile(c.Tiles.Floor)
				}
			}
		}

		planData.Tiles = newTiles
	}

	// 生成された洞窟から部屋領域を抽出
	c.extractCaveRooms(planData)
	return nil
}

// countWallsInRadius は指定半径内の壁タイル数を数える
func (c CaveCellularAutomata) countWallsInRadius(planData *MetaPlan, centerX, centerY, radius int) int {
	wallCount := 0
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	for x := centerX - radius; x <= centerX+radius; x++ {
		for y := centerY - radius; y <= centerY+radius; y++ {
			// 境界外は壁とみなす
			if x < 0 || x >= width || y < 0 || y >= height {
				wallCount++
				continue
			}

			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
			if planData.Tiles[idx].BlockPass {
				wallCount++
			}
		}
	}

	return wallCount
}

// extractCaveRooms は洞窟から部屋領域を抽出する
func (c CaveCellularAutomata) extractCaveRooms(planData *MetaPlan) {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)
	visited := make([]bool, len(planData.Tiles))

	// 連結している床領域を見つけて部屋として登録
	for x := range width {
		for y := range height {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})

			if !planData.Tiles[idx].BlockPass && !visited[idx] {
				// 洪水塗りつぶしで連結領域を見つける
				floorTiles := c.floodFill(planData, x, y, visited)

				// 十分な大きさの領域のみ部屋として認識
				if len(floorTiles) >= c.Row.MinRoomTiles {
					// 境界ボックスを計算
					minX, maxX := x, x
					minY, maxY := y, y

					for _, tilePos := range floorTiles {
						pos := planData.Level.IndexToCoord(gc.TileIdx(tilePos))
						if int(pos.X) < minX {
							minX = int(pos.X)
						}
						if int(pos.X) > maxX {
							maxX = int(pos.X)
						}
						if int(pos.Y) < minY {
							minY = int(pos.Y)
						}
						if int(pos.Y) > maxY {
							maxY = int(pos.Y)
						}
					}

					// 部屋として登録
					room := gc.Rect{Min: consts.Coord[consts.Tile]{X: consts.Tile(minX), Y: consts.Tile(minY)}, Max: consts.Coord[consts.Tile]{X: consts.Tile(maxX), Y: consts.Tile(maxY)}}

					planData.Rooms = append(planData.Rooms, room)
				}
			}
		}
	}
}

// floodFill は洪水塗りつぶしで連結する床タイルを見つける
func (c CaveCellularAutomata) floodFill(planData *MetaPlan, startX, startY int, visited []bool) []int {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)
	var result []int
	var queue [][2]int

	startIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(startX), Y: consts.Tile(startY)})
	queue = append(queue, [2]int{startX, startY})
	visited[startIdx] = true

	// 4方向の隣接タイル
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		x, y := current[0], current[1]
		idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
		result = append(result, int(idx))

		// 隣接タイルをチェック
		for _, dir := range directions {
			nx, ny := x+dir[0], y+dir[1]

			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				nIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(nx), Y: consts.Tile(ny)})

				if !visited[nIdx] && !planData.Tiles[nIdx].BlockPass {
					visited[nIdx] = true
					queue = append(queue, [2]int{nx, ny})
				}
			}
		}
	}

	return result
}

// Widen は床に2マス以上接する壁を確率で床にして通路を広げる
type Widen struct {
	Row   rawschema.WidenStep
	Tiles rawschema.LayoutTiles
}

// PlanMeta は通路を広げる
func (c Widen) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 床タイルの周囲1マスを床にして通路を広げる
	newTiles := make([]rawschema.Tile, len(planData.Tiles))
	copy(newTiles, planData.Tiles)

	for x := 1; x < width-1; x++ {
		for y := 1; y < height-1; y++ {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})

			// 現在が壁で、隣接に床がある場合
			if planData.Tiles[idx].BlockPass {
				adjacentFloorCount := c.countAdjacentFloors(planData, x, y)

				if adjacentFloorCount >= 2 && planData.RNG.Float64() < c.Row.Chance {
					newTiles[idx] = planData.GetTile(c.Tiles.Floor)
				}
			}
		}
	}

	planData.Tiles = newTiles
	return nil
}

// countAdjacentFloors は隣接する床タイルの数を数える
func (c Widen) countAdjacentFloors(planData *MetaPlan, centerX, centerY int) int {
	count := 0
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 4方向の隣接タイルをチェック
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	for _, dir := range directions {
		x, y := centerX+dir[0], centerY+dir[1]

		if x >= 0 && x < width && y >= 0 && y < height {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
			if !planData.Tiles[idx].BlockPass {
				count++
			}
		}
	}

	return count
}

// ScatterWalls は床に確率で壁を撒く
type ScatterWalls struct {
	Row   rawschema.ScatterWallStep
	Tiles rawschema.LayoutTiles
}

// PlanMeta は床に壁を撒く
func (c ScatterWalls) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 床タイルの一部を鍾乳石（壁）に変換
	for x := 2; x < width-2; x++ {
		for y := 2; y < height-2; y++ {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})

			if !planData.Tiles[idx].BlockPass {
				if planData.RNG.Float64() < c.Row.Chance {
					planData.Tiles[idx] = planData.GetTile(c.Tiles.Wall)
				}
			}
		}
	}
	return nil
}

// ConnectRooms は並びの隣の部屋どうしを太いL字の通路で結ぶ
type ConnectRooms struct {
	Tiles rawschema.LayoutTiles
}

// PlanMeta は部屋どうしを結ぶ
func (c ConnectRooms) PlanMeta(planData *MetaPlan) error {
	if len(planData.Rooms) < 2 {
		return nil
	}

	// 各部屋を最低1つの他の部屋と接続する
	for i := range len(planData.Rooms) - 1 {
		room1 := planData.Rooms[i]
		room2 := planData.Rooms[i+1]

		c.createCaveTunnel(planData, room1, room2)
	}
	return nil
}

// createCaveTunnel は2つの洞窟領域間にトンネルを作成する
func (c ConnectRooms) createCaveTunnel(planData *MetaPlan, room1, room2 gc.Rect) {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 各部屋の中心を計算
	center1X := int(room1.Min.X+room1.Max.X) / 2
	center1Y := int(room1.Min.Y+room1.Max.Y) / 2
	center2X := int(room2.Min.X+room2.Max.X) / 2
	center2Y := int(room2.Min.Y+room2.Max.Y) / 2

	// L字型のトンネルを作成（太さ2タイル）
	currentX, currentY := center1X, center1Y

	// 水平方向に移動
	for currentX != center2X {
		if currentX < center2X {
			currentX++
		} else {
			currentX--
		}

		// トンネルを太くする（上下1タイルずつ）
		for dy := -1; dy <= 1; dy++ {
			y := currentY + dy
			if y >= 1 && y < height-1 && currentX >= 1 && currentX < width-1 {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(currentX), Y: consts.Tile(y)})
				planData.Tiles[idx] = planData.GetTile(c.Tiles.Floor)
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

		// トンネルを太くする（左右1タイルずつ）
		for dx := -1; dx <= 1; dx++ {
			x := currentX + dx
			if x >= 1 && x < width-1 && currentY >= 1 && currentY < height-1 {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(currentY)})
				planData.Tiles[idx] = planData.GetTile(c.Tiles.Floor)
			}
		}
	}
}
