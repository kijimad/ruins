package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// BigRoomPlanner は大部屋を生成するビルダー
// マップ全体の大部分を1つの部屋にする
type BigRoomPlanner struct{}

// PlanInitial は初期マップをプランする
func (b BigRoomPlanner) PlanInitial(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// マップの境界を考慮して大きな部屋を1つ作成
	room := gc.Rect{Min: consts.Coord[consts.Tile]{X: consts.Tile(0), Y: consts.Tile(0)}, Max: consts.Coord[consts.Tile]{X: consts.Tile(width - 1), Y: consts.Tile(height - 1)}}

	// 部屋をリストに追加
	planData.Rooms = append(planData.Rooms, room)

	return nil
}

// BigRoomDraw は大部屋を描画し、ランダムにバリエーションを適用するビルダー
type BigRoomDraw struct {
	FloorTile string
	WallTile  string
}

// PlanMeta は大部屋をタイルに描画し、ランダムにバリエーションを適用する
func (b BigRoomDraw) PlanMeta(planData *MetaPlan) error {
	// まず基本の大部屋を描画
	b.drawBasicBigRoom(planData)

	// ランダムにバリエーションを選択して適用
	variantType := planData.RNG.IntN(5)

	switch variantType {
	case 0:
		// 通常の大部屋（何も追加しない）
	case 1:
		// 柱を追加
		b.applyPillars(planData)
	case 2:
		// 障害物を追加
		b.applyObstacles(planData)
	case 3:
		// 迷路パターンを追加
		b.applyMazePattern(planData)
	case 4:
		// 中央台座を追加
		b.applyCenterPlatform(planData)
	}
	return nil
}

// drawBasicBigRoom は基本の大部屋を描画する
func (b BigRoomDraw) drawBasicBigRoom(planData *MetaPlan) {
	for _, room := range planData.Rooms {
		// 部屋の内部を床タイルで埋める
		for x := room.Min.X; x <= room.Max.X; x++ {
			for y := room.Min.Y; y <= room.Max.Y; y++ {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
				planData.Tiles[idx] = planData.GetTile(b.FloorTile)
			}
		}

		// 部屋の境界を壁で囲む
		for y := room.Min.Y; y <= room.Max.Y; y++ {
			// 左辺
			if x := room.Min.X - 1; x >= 0 {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
				if planData.Tiles[idx].Name != b.FloorTile {
					planData.Tiles[idx] = planData.GetTile(b.WallTile)
				}
			}
			// 右辺
			if x := room.Max.X + 1; int(x) < int(planData.Level.TileWidth) {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
				if planData.Tiles[idx].Name != b.FloorTile {
					planData.Tiles[idx] = planData.GetTile(b.WallTile)
				}
			}
		}
		// 上辺と下辺
		for x := room.Min.X; x <= room.Max.X; x++ {
			// 上辺
			if y := room.Min.Y - 1; y >= 0 {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
				if planData.Tiles[idx].Name != b.FloorTile {
					planData.Tiles[idx] = planData.GetTile(b.WallTile)
				}
			}
			// 下辺
			if y := room.Max.Y + 1; int(y) < int(planData.Level.TileHeight) {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: x, Y: y})
				if planData.Tiles[idx].Name != b.FloorTile {
					planData.Tiles[idx] = planData.GetTile(b.WallTile)
				}
			}
		}
	}
}

// applyPillars は部屋に柱を追加する
func (b BigRoomDraw) applyPillars(planData *MetaPlan) {
	// 柱の間隔をランダムに決定（3-6の範囲）
	spacing := 3 + planData.RNG.IntN(4)

	for _, room := range planData.Rooms {
		// 柱の開始位置を計算（部屋の中心から対称に配置）
		startX := int(room.Min.X) + spacing
		startY := int(room.Min.Y) + spacing

		// 規則的に柱を配置
		for x := startX; x < int(room.Max.X); x += spacing + 1 {
			for y := startY; y < int(room.Max.Y); y += spacing + 1 {
				idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
				planData.Tiles[idx] = planData.GetTile("wall")
			}
		}
	}
}

// applyObstacles は部屋にランダムな障害物を追加する
func (b BigRoomDraw) applyObstacles(planData *MetaPlan) {
	for _, room := range planData.Rooms {
		// 障害物の数を部屋のサイズに基づいて決定
		roomWidth := int(room.Width())
		roomHeight := int(room.Height())
		obstacleCount := (roomWidth * roomHeight) / 30 // 面積の1/30程度

		for range obstacleCount {
			// 部屋内のランダムな位置に障害物を配置する
			// IntNの引数が正であることを保証する
			maxXRange := max(1, roomWidth-2)
			maxYRange := max(1, roomHeight-2)
			x := int(room.Min.X) + 1 + planData.RNG.IntN(maxXRange)
			y := int(room.Min.Y) + 1 + planData.RNG.IntN(maxYRange)

			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
			planData.Tiles[idx] = planData.GetTile("wall")
		}
	}
}

// applyMazePattern は部屋に迷路パターンを追加する
func (b BigRoomDraw) applyMazePattern(planData *MetaPlan) {
	for _, room := range planData.Rooms {
		// 格子状に壁を配置し、ランダムに開口部を作る
		// 上端行・下端行には壁を配置しない。上下端への接続性を保証するため
		// 縦の壁を部屋の右端から逆向きに配置
		for x := int(room.Max.X); x >= int(room.Min.X)+2; x -= 3 {
			for y := int(room.Min.Y) + 1; y <= int(room.Max.Y)-1; y++ {
				// 縦の壁を配置（ランダムに開口部を作る）
				if planData.RNG.Float64() > 0.3 {
					idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
					planData.Tiles[idx] = planData.GetTile("wall")
				}
			}
		}

		// 横の壁を部屋の下端から逆向きに配置
		for y := int(room.Max.Y) - 1; y >= int(room.Min.Y)+2; y -= 3 {
			for x := int(room.Min.X); x <= int(room.Max.X); x++ {
				// 横の壁を配置（ランダムに開口部を作る）
				if planData.RNG.Float64() > 0.3 {
					idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
					planData.Tiles[idx] = planData.GetTile("wall")
				}
			}
		}
	}
}

// applyCenterPlatform は部屋に中央台座を追加する
func (b BigRoomDraw) applyCenterPlatform(planData *MetaPlan) {
	for _, room := range planData.Rooms {
		centerX := int(room.Min.X+room.Max.X) / 2
		centerY := int(room.Min.Y+room.Max.Y) / 2

		// 台座のサイズを部屋のサイズに基づいて決定
		platformSize := 2 + planData.RNG.IntN(3) // 2-4タイルの台座

		// 円形の台座を作成
		for dx := -platformSize; dx <= platformSize; dx++ {
			for dy := -platformSize; dy <= platformSize; dy++ {
				distance := dx*dx + dy*dy
				if distance <= platformSize*platformSize {
					x := centerX + dx
					y := centerY + dy
					if x >= int(room.Min.X) && x <= int(room.Max.X) &&
						y >= int(room.Min.Y) && y <= int(room.Max.Y) {
						idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
						// 外周は壁、内部は床のまま
						if distance >= (platformSize-1)*(platformSize-1) {
							planData.Tiles[idx] = planData.GetTile("wall")
						}
					}
				}
			}
		}
	}
}
