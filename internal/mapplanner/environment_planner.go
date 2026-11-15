package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// EnvironmentPlanner は環境情報を計算するプランナー
type EnvironmentPlanner struct{}

// PlanMeta はタイルごとの環境を計算して MetaPlan.Environments に入れる。
// タイルの行から決まる値はそのまま使い、水の影響は周りのタイルから導く
func (p EnvironmentPlanner) PlanMeta(mp *MetaPlan) error {
	mp.Environments = make([]gc.TileEnvironment, len(mp.Tiles))
	for i := range mp.Tiles {
		mp.Environments[i] = raw.TileEnvironmentOf(mp.Tiles[i])
		mp.Environments[i].Water = p.calcWater(mp, i)
	}

	return nil
}

// calcWater は水の影響を計算する。水のタイルの上なら水中、隣が水のタイルなら水辺、どちらでもなければゼロ値
func (p EnvironmentPlanner) calcWater(mp *MetaPlan, idx int) rawschema.WaterExposure {
	if isWaterTile(mp.Tiles[idx]) {
		return rawschema.Submerged
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
			return rawschema.Nearby
		}
	}

	return ""
}

// isWaterTile はタイルが水のタイルかを返す
func isWaterTile(tile rawschema.Tile) bool {
	return tile.Water != nil && *tile.Water
}
