package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// DoorPlanner は部屋の入口にランダムにドアを配置するプランナー。
// 床タイルの左右または上下が壁であるパターンを検出してドアを置く
type DoorPlanner struct {
	DoorChance float64 // ドア生成確率（0.0〜1.0）
}

// PlanMeta は全タイルを走査してドアを配置する
func (p DoorPlanner) PlanMeta(mp *MetaPlan) error {
	width := int(mp.Level.TileWidth)
	height := int(mp.Level.TileHeight)

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			idx := y*width + x
			if mp.Tiles[idx].BlockPass {
				continue
			}

			if !isDoorPattern(mp, x, y, width) {
				continue
			}

			if mp.RNG.Float64() >= p.DoorChance {
				continue
			}

			mp.Doors = append(mp.Doors, DoorSpec{
				Coord:       consts.Coord[int]{X: x, Y: y},
				Orientation: doorOrientation(mp, x, y, width),
			})
		}
	}
	return nil
}

// isDoorPattern は床タイルがドア配置可能なパターンかを判定する。
// 左右が壁かつ上下が床、または上下が壁かつ左右が床の場合にtrueを返す
func isDoorPattern(mp *MetaPlan, x, y, width int) bool {
	idx := y*width + x
	left := mp.Tiles[idx-1]
	right := mp.Tiles[idx+1]
	top := mp.Tiles[idx-width]
	bottom := mp.Tiles[idx+width]

	// 左右が壁、上下が床 → 縦向きドア
	if left.BlockPass && right.BlockPass && !top.BlockPass && !bottom.BlockPass {
		return true
	}
	// 上下が壁、左右が床 → 横向きドア
	if top.BlockPass && bottom.BlockPass && !left.BlockPass && !right.BlockPass {
		return true
	}

	return false
}

// doorOrientation はドア位置の隣接タイルから向きを判定する
func doorOrientation(mp *MetaPlan, x, y, width int) gc.DoorOrientation {
	idx := y*width + x
	if mp.Tiles[idx-1].BlockPass && mp.Tiles[idx+1].BlockPass {
		return gc.DoorOrientationVertical
	}
	return gc.DoorOrientationHorizontal
}
