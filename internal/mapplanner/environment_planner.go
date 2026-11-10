package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/oapi"
)

// EnvironmentPlanner は環境情報を計算するプランナー
type EnvironmentPlanner struct{}

// PlanMeta はタイルごとの環境を計算して MetaPlan.Environments に入れる。
// 囲われと水の影響は周りのタイルから導き、植生はタイルの行の値をそのまま使う
func (p EnvironmentPlanner) PlanMeta(mp *MetaPlan) error {
	outdoor := p.floodFillOutdoor(mp)

	mp.Environments = make([]gc.TileEnvironment, len(mp.Tiles))
	for i := range mp.Tiles {
		env := &mp.Environments[i]
		env.Shelter = oapi.Outdoor
		if !outdoor[i] {
			env.Shelter = oapi.Indoor
		}
		env.Water = p.calcWater(mp, i)
		if mp.Tiles[i].Foliage != nil {
			env.Foliage = *mp.Tiles[i].Foliage
		}
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

// calcWater は水の影響を計算する。水のタイルの上なら水中、隣が水のタイルなら水辺、どちらでもなければゼロ値
func (p EnvironmentPlanner) calcWater(mp *MetaPlan, idx int) oapi.WaterExposure {
	if isWaterTile(mp.Tiles[idx]) {
		return oapi.Submerged
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
		if isWaterTile(mp.Tiles[adjIdx]) {
			return oapi.Nearby
		}
	}

	return ""
}

// isWaterTile はタイルが水のタイルかを返す
func isWaterTile(tile oapi.Tile) bool {
	return tile.Water != nil && *tile.Water
}
