package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

// CavePlanner は洞窟風レイアウトを生成するビルダー
// セルラーオートマトンを使用して有機的な洞窟形状を作成
// TODO: 全然わかってないので理解する
type CavePlanner struct{}

// PlanInitial は初期洞窟マップをプランする
func (c CavePlanner) PlanInitial(planData *MetaPlan) error {
	// 初期状態をランダムに設定（30%の確率で壁、より広い空間を確保）
	for i := range planData.Tiles {
		if planData.RNG.Float64() < 0.30 {
			planData.Tiles[i] = planData.GetTile("wall")
		} else {
			planData.Tiles[i] = planData.GetTile("floor")
		}
	}

	return nil
}

// CaveCellularAutomata はセルラーオートマトンによる洞窟生成
type CaveCellularAutomata struct {
	// 反復回数
	Iterations int
}

// PlanMeta はセルラーオートマトンで洞窟を生成する
func (c CaveCellularAutomata) PlanMeta(planData *MetaPlan) error {
	iterations := c.Iterations
	if iterations <= 0 {
		iterations = 5 // デフォルト反復回数
	}

	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// セルラーオートマトンを指定回数実行
	for iter := 0; iter < iterations; iter++ {
		newTiles := make([]raw.TileRaw, len(planData.Tiles))

		for x := 0; x < width; x++ {
			for y := 0; y < height; y++ {
				idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

				// 左右端は壁にする
				// 上下端はオートマトンのルールに任せる
				if x == 0 || x == width-1 {
					newTiles[idx] = planData.GetTile("wall")
					continue
				}

				// 周囲の壁の数を数える
				wallCount := c.countWallsInRadius(planData, x, y, 1)

				// ルール：周囲に6つ以上の壁があれば壁、そうでなければ床（より通路を確保）
				if wallCount >= 6 {
					newTiles[idx] = planData.GetTile("wall")
				} else {
					newTiles[idx] = planData.GetTile("floor")
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

			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
			if !planData.Tiles[idx].Walkable {
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
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

			if planData.Tiles[idx].Walkable && !visited[idx] {
				// 洪水塗りつぶしで連結領域を見つける
				floorTiles := c.floodFill(planData, x, y, visited)

				// 十分な大きさの領域のみ部屋として認識
				if len(floorTiles) >= 10 {
					// 境界ボックスを計算
					minX, maxX := x, x
					minY, maxY := y, y

					for _, tilePos := range floorTiles {
						tileX, tileY := planData.Level.XYTileCoord(resources.TileIdx(tilePos))
						if int(tileX) < minX {
							minX = int(tileX)
						}
						if int(tileX) > maxX {
							maxX = int(tileX)
						}
						if int(tileY) < minY {
							minY = int(tileY)
						}
						if int(tileY) > maxY {
							maxY = int(tileY)
						}
					}

					// 部屋として登録
					room := gc.Rect{
						X1: gc.Tile(minX),
						Y1: gc.Tile(minY),
						X2: gc.Tile(maxX),
						Y2: gc.Tile(maxY),
					}
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

	startIdx := planData.Level.XYTileIndex(gc.Tile(startX), gc.Tile(startY))
	queue = append(queue, [2]int{startX, startY})
	visited[startIdx] = true

	// 4方向の隣接タイル
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		x, y := current[0], current[1]
		idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
		result = append(result, int(idx))

		// 隣接タイルをチェック
		for _, dir := range directions {
			nx, ny := x+dir[0], y+dir[1]

			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				nIdx := planData.Level.XYTileIndex(gc.Tile(nx), gc.Tile(ny))

				if !visited[nIdx] && planData.Tiles[nIdx].Walkable {
					visited[nIdx] = true
					queue = append(queue, [2]int{nx, ny})
				}
			}
		}
	}

	return result
}

// CavePathWidener は通路を広げる
type CavePathWidener struct{}

// PlanMeta は狭い通路を広げる
func (c CavePathWidener) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 床タイルの周囲1マスを床にして通路を広げる
	newTiles := make([]raw.TileRaw, len(planData.Tiles))
	copy(newTiles, planData.Tiles)

	for x := 1; x < width-1; x++ {
		for y := 1; y < height-1; y++ {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

			// 現在が壁で、隣接に床がある場合
			if !planData.Tiles[idx].Walkable {
				adjacentFloorCount := c.countAdjacentFloors(planData, x, y)

				// 隣接する床が2個以上ある場合、30%の確率で床にする
				if adjacentFloorCount >= 2 && planData.RNG.Float64() < 0.3 {
					newTiles[idx] = planData.GetTile("floor")
				}
			}
		}
	}

	planData.Tiles = newTiles
	return nil
}

// countAdjacentFloors は隣接する床タイルの数を数える
func (c CavePathWidener) countAdjacentFloors(planData *MetaPlan, centerX, centerY int) int {
	count := 0
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 4方向の隣接タイルをチェック
	directions := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	for _, dir := range directions {
		x, y := centerX+dir[0], centerY+dir[1]

		if x >= 0 && x < width && y >= 0 && y < height {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
			if planData.Tiles[idx].Walkable {
				count++
			}
		}
	}

	return count
}

// CaveStalactites は鍾乳石や岩の障害物を配置する
type CaveStalactites struct{}

// PlanMeta は洞窟内に鍾乳石を配置する
func (c CaveStalactites) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 床タイルの一部を鍾乳石（壁）に変換
	for x := 2; x < width-2; x++ {
		for y := 2; y < height-2; y++ {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

			if planData.Tiles[idx].Walkable {
				// 2%の確率で鍾乳石を配置（確率を下げてより通行可能に）
				if planData.RNG.Float64() < 0.02 {
					planData.Tiles[idx] = planData.GetTile("wall")
				}
			}
		}
	}
	return nil
}

// CaveConnector は隔離された洞窟領域を接続する
type CaveConnector struct{}

// PlanMeta は隔離された洞窟領域を接続する
func (c CaveConnector) PlanMeta(planData *MetaPlan) error {
	if len(planData.Rooms) < 2 {
		return nil
	}

	// 各部屋を最低1つの他の部屋と接続する
	for i := 0; i < len(planData.Rooms)-1; i++ {
		room1 := planData.Rooms[i]
		room2 := planData.Rooms[i+1]

		c.createCaveTunnel(planData, room1, room2)
	}
	return nil
}

// createCaveTunnel は2つの洞窟領域間にトンネルを作成する
func (c CaveConnector) createCaveTunnel(planData *MetaPlan, room1, room2 gc.Rect) {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 各部屋の中心を計算
	center1X := int(room1.X1+room1.X2) / 2
	center1Y := int(room1.Y1+room1.Y2) / 2
	center2X := int(room2.X1+room2.X2) / 2
	center2Y := int(room2.Y1+room2.Y2) / 2

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
				idx := planData.Level.XYTileIndex(gc.Tile(currentX), gc.Tile(y))
				planData.Tiles[idx] = planData.GetTile("floor")
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
				idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(currentY))
				planData.Tiles[idx] = planData.GetTile("floor")
			}
		}
	}
}

// NewCavePlanner は洞窟ビルダーを作成する
func NewCavePlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(CavePlanner{})
	chain.With(CaveCellularAutomata{Iterations: 3}) // セルラーオートマトン
	chain.With(CavePathWidener{})                   // 通路を広げる
	chain.With(CaveConnector{})                     // 隔離領域を接続
	chain.With(CaveStalactites{})                   // 鍾乳石配置

	return chain, nil
}
