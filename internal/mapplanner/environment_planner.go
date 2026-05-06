package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
)

// EnvironmentPlanner は環境情報を計算するプランナー
type EnvironmentPlanner struct{}

// PlanMeta は環境情報を計算して TileRaw に設定する
func (p EnvironmentPlanner) PlanMeta(mp *MetaPlan) error {
	// 屋外タイルを特定する
	outdoor := p.floodFillOutdoor(mp)

	// 各タイルの環境情報を設定する
	for i := range mp.Tiles {
		if outdoor[i] {
			mp.Tiles[i].Shelter = int32(gc.ShelterNone)
		} else {
			mp.Tiles[i].Shelter = int32(gc.ShelterFull)
		}

		// Water: 隣接する水タイルから計算
		mp.Tiles[i].Water = int32(p.calcWater(mp, i))

		// Foliage: タイル名から計算
		mp.Tiles[i].Foliage = int32(p.calcFoliage(mp, i))
	}

	return nil
}

// floodFillOutdoor はマップ端から到達可能なタイルを屋外としてマーク
func (p EnvironmentPlanner) floodFillOutdoor(mp *MetaPlan) []bool {
	outdoor := make([]bool, len(mp.Tiles))
	visited := make([]bool, len(mp.Tiles))
	width := int(mp.Level.TileWidth)
	height := int(mp.Level.TileHeight)

	// BFS キュー（マップ端から開始）
	queue := []int{}

	// 上端と下端
	for x := 0; x < width; x++ {
		queue = append(queue, x)                  // 上端
		queue = append(queue, (height-1)*width+x) // 下端
	}
	// 左端と右端（上下端を除く）
	for y := 1; y < height-1; y++ {
		queue = append(queue, y*width)         // 左端
		queue = append(queue, y*width+width-1) // 右端
	}

	// BFS: 壁以外を辿り、到達可能な領域を屋外としてマーク
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		if idx < 0 || idx >= len(mp.Tiles) || visited[idx] {
			continue
		}
		visited[idx] = true

		if mp.Tiles[idx].BlockPass {
			continue // 壁は通過しない
		}

		outdoor[idx] = true

		// 4方向の隣接タイルをキューに追加
		x, y := idx%width, idx/width
		if x > 0 {
			queue = append(queue, idx-1)
		}
		if x < width-1 {
			queue = append(queue, idx+1)
		}
		if y > 0 {
			queue = append(queue, idx-width)
		}
		if y < height-1 {
			queue = append(queue, idx+width)
		}
	}

	return outdoor
}

// calcWater は隣接タイルから水の影響を計算する
func (p EnvironmentPlanner) calcWater(mp *MetaPlan, idx int) gc.WaterType {
	// 現在のタイルが水タイルかチェック
	if isWaterTile(mp.Tiles[idx].Name) {
		return gc.WaterSubmerged
	}

	// 隣接タイルに水があるかチェック
	width := int(mp.Level.TileWidth)
	height := int(mp.Level.TileHeight)
	x, y := idx%width, idx/width

	adjacentIndices := []int{}
	if x > 0 {
		adjacentIndices = append(adjacentIndices, idx-1)
	}
	if x < width-1 {
		adjacentIndices = append(adjacentIndices, idx+1)
	}
	if y > 0 {
		adjacentIndices = append(adjacentIndices, idx-width)
	}
	if y < height-1 {
		adjacentIndices = append(adjacentIndices, idx+width)
	}

	for _, adjIdx := range adjacentIndices {
		if isWaterTile(mp.Tiles[adjIdx].Name) {
			return gc.WaterNearby
		}
	}

	return gc.WaterNone
}

// calcFoliage はタイル名から植生の影響を計算する
// TODO: 名前ではなくタイルの属性で判定する
func (p EnvironmentPlanner) calcFoliage(mp *MetaPlan, idx int) gc.FoliageType {
	name := mp.Tiles[idx].Name

	switch name {
	case "forest", "tree":
		return gc.FoliageForest
	case "grass":
		return gc.FoliageGrass
	default:
		return gc.FoliageNone
	}
}

// isWaterTile はタイル名が水タイルかを判定する
// TODO: 名前ではなくタイルの属性で判定する
func isWaterTile(name string) bool {
	switch name {
	case "water", "deep_water", "river", "pond":
		return true
	default:
		return false
	}
}
