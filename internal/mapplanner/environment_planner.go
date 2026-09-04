package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/oapi"
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
			mp.Tiles[i].Shelter = oapi.ShelterType(gc.ShelterNone)
		} else {
			mp.Tiles[i].Shelter = oapi.ShelterType(gc.ShelterFull)
		}

		// Water: 隣接する水タイルから計算
		mp.Tiles[i].Water = oapi.WaterType(p.calcWater(mp, i))

		// Foliage: タイル名から計算
		mp.Tiles[i].Foliage = oapi.FoliageType(p.calcFoliage(mp, i))
	}

	return nil
}

// floodFillOutdoor はマップ外周に達する連結領域を屋外としてマークする。
// 囲われの判定は geometry.EnclosedRegion に集約し、扉開閉時の再計算と意味論を共有する
func (p EnvironmentPlanner) floodFillOutdoor(mp *MetaPlan) []bool {
	width := int(mp.Level.TileWidth)
	height := int(mp.Level.TileHeight)
	blocked := func(x, y int) bool { return mp.Tiles[y*width+x].BlockPass }

	outdoor := make([]bool, len(mp.Tiles))
	visited := make([]bool, len(mp.Tiles))
	for i := range mp.Tiles {
		if visited[i] || mp.Tiles[i].BlockPass {
			continue
		}
		cells, touchesEdge := geometry.EnclosedRegion(width, height, blocked, i%width, i/width)
		for _, c := range cells {
			visited[c] = true
			outdoor[c] = touchesEdge
		}
	}
	return outdoor
}

// calcWater は隣接タイルから水の影響を計算する
func (p EnvironmentPlanner) calcWater(mp *MetaPlan, idx int) gc.WaterType {
	// 現在のタイルが水タイルかチェック
	if isWaterTile(mp.Tiles[idx].Id) {
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
		if isWaterTile(mp.Tiles[adjIdx].Id) {
			return gc.WaterNearby
		}
	}

	return gc.WaterNone
}

// calcFoliage はタイル名から植生の影響を計算する
// TODO: 名前ではなくタイルの属性で判定する
func (p EnvironmentPlanner) calcFoliage(mp *MetaPlan, idx int) gc.FoliageType {
	name := mp.Tiles[idx].Id

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
