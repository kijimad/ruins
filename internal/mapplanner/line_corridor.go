package mapplanner

import (
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
)

// LineCorridorPlanner は直線廊下を生成するビルダー。
// 廊下は3タイル幅で生成され、部屋との接続部は1タイル幅に狭まる
type LineCorridorPlanner struct{}

// PlanMeta はメタデータをビルドする
func (b LineCorridorPlanner) PlanMeta(planData *MetaPlan) error {
	b.BuildCorridors(planData)
	return nil
}

// BuildCorridors は廊下をビルドする
func (b LineCorridorPlanner) BuildCorridors(planData *MetaPlan) {
	// 接続済みの部屋。通路を2重に計算しないようにする
	connected := map[int]bool{}
	// 廊下のスライス
	for i, room := range planData.Rooms {
		roomDistances := map[int]float64{}
		centerX, centerY := room.Center()
		for j, otherRoom := range planData.Rooms {
			isExist := connected[j]
			if i != j && !isExist {
				oCenterX, oCenterY := otherRoom.Center()
				distance := geometry.Distance(float64(centerX), float64(centerY), float64(oCenterX), float64(oCenterY))
				roomDistances[j] = float64(distance)
			}
		}

		if len(roomDistances) > 0 {
			closestIdx := -1
			closestDist := math.MaxFloat64
			for k, v := range roomDistances {
				if v < closestDist {
					closestDist = v
					closestIdx = k
				}
			}
			destCenterX, destCenterY := planData.Rooms[closestIdx].Center()

			// 中心線のL字型廊下を生成する
			centerStart := consts.Coord[consts.Tile]{X: centerX, Y: centerY}
			centerEnd := consts.Coord[consts.Tile]{X: destCenterX, Y: destCenterY}
			centerPoints := createLShapedCorridor(centerStart, centerEnd)

			// サイドのL字型廊下を生成する
			const corridorWidth = 3
			var sidePoints []consts.Coord[consts.Tile]
			for offsetX := -(corridorWidth / 2); offsetX <= corridorWidth/2; offsetX++ {
				for offsetY := -(corridorWidth / 2); offsetY <= corridorWidth/2; offsetY++ {
					if offsetX == 0 && offsetY == 0 {
						continue
					}
					start := consts.Coord[consts.Tile]{X: centerX + consts.Tile(offsetX), Y: centerY + consts.Tile(offsetY)}
					end := consts.Coord[consts.Tile]{X: destCenterX + consts.Tile(offsetX), Y: destCenterY + consts.Tile(offsetY)}
					sidePoints = append(sidePoints, createLShapedCorridor(start, end)...)
				}
			}

			corridor := make([]gc.TileIdx, 0, len(centerPoints)+len(sidePoints))

			// 中心線は無条件に床に変換する
			for _, p := range centerPoints {
				idx := planData.Level.XYTileIndex(p.X, p.Y)
				if isValidTileIdx(planData, idx) && planData.Tiles[idx].Name == consts.TileNameWall {
					planData.Tiles[idx] = planData.GetTile(consts.TileNameFloor)
				}
				corridor = append(corridor, idx)
			}

			// サイドは部屋に隣接するタイルをスキップして、接続部を1タイル幅に狭める
			for _, p := range sidePoints {
				idx := planData.Level.XYTileIndex(p.X, p.Y)
				if isValidTileIdx(planData, idx) && planData.Tiles[idx].Name == consts.TileNameWall {
					if isAdjacentToRoom(planData.Rooms, int(p.X), int(p.Y)) {
						continue
					}
					planData.Tiles[idx] = planData.GetTile(consts.TileNameFloor)
				}
				corridor = append(corridor, idx)
			}

			planData.Corridors = append(planData.Corridors, corridor)
		}
		connected[i] = true
	}
}

func isValidTileIdx(planData *MetaPlan, idx gc.TileIdx) bool {
	return int(idx) >= 0 && int(idx) < int(planData.Level.TileWidth)*int(planData.Level.TileHeight)
}

// isAdjacentToRoom はタイルが部屋内または部屋に隣接しているかを判定する。
// 部屋の矩形を上下左右に1タイル膨張させた範囲に含まれるかで判定する。
// 対角方向も含めることで、部屋の角付近でサイドタイルが部屋壁を上書きするのを防ぐ
func isAdjacentToRoom(rooms []gc.Rect, x, y int) bool {
	for _, room := range rooms {
		if x >= int(room.X1)-1 && x <= int(room.X2)+1 && y >= int(room.Y1)-1 && y <= int(room.Y2)+1 {
			return true
		}
	}
	return false
}

// createLShapedCorridor は横と縦のみのL字型廊下を生成する
func createLShapedCorridor(start, end consts.Coord[consts.Tile]) []consts.Coord[consts.Tile] {
	var points []consts.Coord[consts.Tile]

	// 開始点から水平に移動
	current := start
	if start.X < end.X {
		for current.X <= end.X {
			points = append(points, current)
			if current.X == end.X {
				break
			}
			current.X++
		}
	} else {
		for current.X >= end.X {
			points = append(points, current)
			if current.X == end.X {
				break
			}
			current.X--
		}
	}

	// 水平移動後の位置から垂直に移動
	if start.Y < end.Y {
		for current.Y < end.Y {
			current.Y++
			points = append(points, current)
		}
	} else if start.Y > end.Y {
		for current.Y > end.Y {
			current.Y--
			points = append(points, current)
		}
	}

	return points
}
