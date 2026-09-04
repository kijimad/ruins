package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// shelterRecalcRadius は囲われを再計算する範囲の半径。建物ひとつが余裕をもって
// 収まる大きさにする。市街地の建物は1チャンク24タイルの区画に収まる
const shelterRecalcRadius = 32

// RecalcShelterAround は座標の周囲でタイルの囲われ Shelter を再計算する。
// 扉の開閉のような通行可否の構造変化後に呼ぶ。座標とその4隣接を種に、
// 壁と閉じた扉を越えずに繋がる領域を求めて屋内外を書き直す。
//
// 再計算範囲の外周へ達した領域は原則として屋外だが、領域内に既存の屋外タイルを
// ひとつも見なかった場合は範囲より大きい囲いと区別できないため書き換えない。
// この保守則により、範囲に収まらない構造は誤って冷やされず現状維持で外れる。
func RecalcShelterAround(world w.World, x, y consts.Tile) {
	const size = shelterRecalcRadius*2 + 1
	minX := int(x) - shelterRecalcRadius
	minY := int(y) - shelterRecalcRadius

	// 範囲内の通行不可と書き込み先タイルを集める。閉じた扉は BlockPass を持つので壁と同様に領域を区切る
	blocked := make([]bool, size*size)
	blockQuery := query.ActiveFilter2[gc.GridElement, gc.BlockPass](world).Query()
	for blockQuery.Next() {
		entity := blockQuery.Entity()
		if world.Components.Dead.Has(entity) {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		lx, ly := int(grid.X)-minX, int(grid.Y)-minY
		if lx >= 0 && lx < size && ly >= 0 && ly < size {
			blocked[ly*size+lx] = true
		}
	}

	tiles := make(map[int]ecs.Entity)
	tileQuery := query.ActiveFilter2[gc.GridElement, gc.TileEnvironment](world).Query()
	for tileQuery.Next() {
		entity := tileQuery.Entity()
		if world.Components.Dead.Has(entity) {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		lx, ly := int(grid.X)-minX, int(grid.Y)-minY
		if lx >= 0 && lx < size && ly >= 0 && ly < size {
			tiles[ly*size+lx] = entity
		}
	}

	blockedAt := func(bx, by int) bool { return blocked[by*size+bx] }

	// 扉が開けば両側は1つの領域として、閉じれば分断された各側が別領域として書き直される。
	// 閉じた扉は BlockPass を持つので中心の種は弾かれるが、4隣接の種が室内外の両側を別々に拾う
	visited := make([]bool, size*size)
	seeds := [][2]int{
		{shelterRecalcRadius, shelterRecalcRadius},
		{shelterRecalcRadius - 1, shelterRecalcRadius},
		{shelterRecalcRadius + 1, shelterRecalcRadius},
		{shelterRecalcRadius, shelterRecalcRadius - 1},
		{shelterRecalcRadius, shelterRecalcRadius + 1},
	}
	for _, s := range seeds {
		if visited[s[1]*size+s[0]] || blockedAt(s[0], s[1]) {
			continue
		}
		cells, touchesEdge := geometry.EnclosedRegion(size, size, blockedAt, s[0], s[1])

		sawOutdoor := false
		for _, c := range cells {
			visited[c] = true
			if tile, ok := tiles[c]; ok {
				if world.Components.TileEnvironment.Get(tile).Shelter == gc.ShelterNone {
					sawOutdoor = true
				}
			}
		}

		shelter := gc.ShelterFull
		switch {
		case touchesEdge && sawOutdoor:
			shelter = gc.ShelterNone
		case touchesEdge:
			// 範囲より大きい囲いか判定できないので現状維持にする
			continue
		}
		for _, c := range cells {
			if tile, ok := tiles[c]; ok {
				world.Components.TileEnvironment.Get(tile).Shelter = shelter
			}
		}
	}
}
