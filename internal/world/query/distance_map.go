package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetConvergeDistanceMap はシングルトンから、動き方が converge の敵が下る RaidTargets の的からの距離の地図を取得する。未構築なら作る。手番の外で引くと
// 前の手番の地図が返る
func GetConvergeDistanceMap(world w.World) *gc.DistanceMap {
	m := GetSingleton[gc.ConvergeDistanceMap](world, world.Components.ConvergeDistanceMap)
	if m == nil {
		return nil
	}
	if !m.Built {
		buildDistanceMap(world, &m.DistanceMap, RaidTargets(world), convergeRule)
	}
	return &m.DistanceMap
}

// InvalidateDistanceMaps は寄り集まる敵とついて行く者の距離の地図を無効化する。空間の索引と違い、物が動くたびには呼ばず、敵の
// 手番の初めと波を湧かせる前にだけ呼ぶ。手番の間は古い地図を全員で下る
func InvalidateDistanceMaps(world w.World) {
	if m := GetSingleton[gc.ConvergeDistanceMap](world, world.Components.ConvergeDistanceMap); m != nil {
		m.Built = false
	}
	if m := GetSingleton[gc.FollowDistanceMap](world, world.Components.FollowDistanceMap); m != nil {
		m.Built = false
	}
}

// GetFollowDistanceMap はシングルトンから、ついて行く者が下るプレイヤーからの距離の地図を取得する。未構築なら作る。
// ついて行く者は置物を壊さないので HP を持つ置物も通れないマスとして数え、味方を含む他のキャラクターのマスは重く数えて
// 回り込ませる
func GetFollowDistanceMap(world w.World) *gc.DistanceMap {
	m := GetSingleton[gc.FollowDistanceMap](world, world.Components.FollowDistanceMap)
	if m == nil {
		return nil
	}
	if !m.Built {
		var targets []consts.Coord[consts.Tile]
		if player, err := GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(player) {
			targets = append(targets, world.Components.GridElement.Get(player).Coord)
		}
		buildDistanceMap(world, &m.DistanceMap, targets, followRule)
	}
	return &m.DistanceMap
}

// distanceRule は距離の地図の重みの規則。重みを Spread の前提の 1 か gc.DistanceMapObstacleCost か通れないかに閉じるため、
// 数でなく真偽で選ぶ
type distanceRule struct {
	// breakProps なら HP を持つ通れない置物を壊して通るものとして重く数え、偽なら通れないマスとして数える
	breakProps bool
	// avoidAllies なら味方のマスも重く数えて回り込ませ、偽なら空いたマスと同じに数える
	avoidAllies bool
}

var (
	// convergeRule は動き方が converge の敵の規則。置物を壊して進み、味方同士は通り抜けて寄り集まる
	convergeRule = distanceRule{breakProps: true}
	// followRule はついて行く者の規則。置物を壊さず、仲間を回り込む
	followRule = distanceRule{avoidAllies: true}
)

// buildDistanceMap は空間の索引と置物から各マスの重みを書き出し、的 targets から距離を広げる。通れるマスは重み1、味方でない
// キャラクターのマスは重み gc.DistanceMapObstacleCost で通れるものとして数える。HP を持つ通れない置物と味方のマスは
// rule で決める。HP を持たず通れない壁や地形は数えない
func buildDistanceMap(world w.World, m *gc.DistanceMap, targets []consts.Coord[consts.Tile], rule distanceRule) {
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
	if rule.breakProps {
		q := ActiveFilter3[gc.Prop, gc.HP, gc.GridElement](world).Query()
		for q.Next() {
			_, _, g := q.Get()
			if m.InBounds(g.Coord) && m.Blocked[m.Index(g.Coord)] {
				m.Cost[m.Index(g.Coord)] = gc.DistanceMapObstacleCost
			}
		}
	}
	for g, e := range si.Characters {
		if !m.InBounds(g.Coord) || m.Blocked[m.Index(g.Coord)] {
			continue
		}
		if IsAlly(world, e) && !rule.avoidAllies {
			continue
		}
		m.Cost[m.Index(g.Coord)] = gc.DistanceMapObstacleCost
	}
	m.Spread(targets)
}
