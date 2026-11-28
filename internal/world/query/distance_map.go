package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetDistanceMap はシングルトンから、RaidTargets の的からの距離の地図を取得する。未構築なら作る。手番の外で引くと
// 前の手番の地図が返る
func GetDistanceMap(world w.World) *gc.DistanceMap {
	m := GetSingleton[gc.DistanceMap](world, world.Components.DistanceMap)
	if m == nil {
		return nil
	}
	if !m.Built {
		buildDistanceMap(world, m, RaidTargets(world))
	}
	return m
}

// InvalidateDistanceMap は距離の地図を無効化する。空間の索引と違い、物が動くたびには呼ばず、敵の手番の初めと波を
// 湧かせる前にだけ呼ぶ。手番の間は古い地図を全員で下る
func InvalidateDistanceMap(world w.World) {
	if m := GetSingleton[gc.DistanceMap](world, world.Components.DistanceMap); m != nil {
		m.Built = false
	}
}

// buildDistanceMap は空間の索引と置物から各マスの重みを書き出し、的 targets から距離を広げる。通れるマスと、プレイヤーと
// 味方のマスは重み1、HP を持つ通れない置物と、それ以外のキャラクターのマスは重み gc.DistanceMapObstacleCost で
// 通れるものとして数える。HP を持たず通れない壁や地形は数えない
func buildDistanceMap(world w.World, m *gc.DistanceMap, targets []consts.Coord[consts.Tile]) {
	m.Built = true
	si := GetSpatialIndex(world)
	if si == nil {
		m.Reset(0, 0)
		return
	}
	m.Reset(si.MapWidth, si.MapHeight)
	for g, blocked := range si.Blocked {
		if !blocked || !m.InBounds(g.Coord) {
			continue
		}
		i := m.Index(g.Coord)
		m.Cost[i] = 0
		m.Blocked[i] = true
	}
	q := ActiveFilter3[gc.Prop, gc.HP, gc.GridElement](world).Query()
	for q.Next() {
		_, _, g := q.Get()
		if m.InBounds(g.Coord) && m.Blocked[m.Index(g.Coord)] {
			m.Cost[m.Index(g.Coord)] = gc.DistanceMapObstacleCost
		}
	}
	for g, e := range si.Characters {
		if !m.InBounds(g.Coord) || m.Blocked[m.Index(g.Coord)] || IsAlly(world, e) {
			continue
		}
		m.Cost[m.Index(g.Coord)] = gc.DistanceMapObstacleCost
	}
	m.Spread(targets)
}
