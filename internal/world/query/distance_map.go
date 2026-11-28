package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetCubeDistanceMap はシングルトンから、RaidTargets の的のキューブからの距離の地図を取得する。未構築なら作る。手番の外で引くと
// 前の手番の地図が返る
func GetCubeDistanceMap(world w.World) *gc.DistanceMap {
	m := GetSingleton[gc.CubeDistanceMap](world, world.Components.CubeDistanceMap)
	if m == nil {
		return nil
	}
	if !m.Built {
		buildDistanceMap(world, &m.DistanceMap, RaidTargets(world), cubeRule)
	}
	return &m.DistanceMap
}

// InvalidateDistanceMaps はキューブとプレイヤーからの距離の地図を無効化する。空間の索引と違い、物が動くたびには呼ばず、敵の
// 手番の初めと波を湧かせる前にだけ呼ぶ。手番の間は古い地図を全員で下る
func InvalidateDistanceMaps(world w.World) {
	if m := GetSingleton[gc.CubeDistanceMap](world, world.Components.CubeDistanceMap); m != nil {
		m.Built = false
	}
	if m := GetSingleton[gc.PlayerDistanceMap](world, world.Components.PlayerDistanceMap); m != nil {
		m.Built = false
	}
}

// GetPlayerDistanceMap はシングルトンから、プレイヤーからの距離の地図を取得する。未構築なら作る。手番の外で引くと前の
// 手番の地図が返る
func GetPlayerDistanceMap(world w.World) *gc.DistanceMap {
	m := GetSingleton[gc.PlayerDistanceMap](world, world.Components.PlayerDistanceMap)
	if m == nil {
		return nil
	}
	if !m.Built {
		var targets []consts.Coord[consts.Tile]
		if player, err := GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(player) {
			targets = append(targets, world.Components.GridElement.Get(player).Coord)
		}
		buildDistanceMap(world, &m.DistanceMap, targets, playerRule)
	}
	return &m.DistanceMap
}

// distanceRule は距離の地図の重みの規則。重みを Spread の前提の 1 か gc.DistanceMapObstacleCost か通れないかに閉じるため、
// 数でなく真偽で選ぶ
type distanceRule struct {
	// breakProps なら HP を持つ通れない置物のマスを重み gc.DistanceMapObstacleCost で通れるものにする
	breakProps bool
	// avoidAllies なら FactionAlly のプレイヤー側のマスも重み gc.DistanceMapObstacleCost にし、偽なら重み1にする
	avoidAllies bool
	// blockDeployed なら展開して動かないキューブのマスを通れないものにする。重みにすると、回り道が長いときに後ろで待ち続ける
	blockDeployed bool
}

var (
	// cubeRule は襲撃の敵の規則。置物を壊して進み、殴る相手のプレイヤー側のマスは避けない
	cubeRule = distanceRule{breakProps: true}
	// playerRule はついて行く者の規則。置物を壊さず、プレイヤー側のマスも回り込み、展開したキューブは通れない
	playerRule = distanceRule{avoidAllies: true, blockDeployed: true}
)

// buildDistanceMap は各マスの重みを書き出し、的 targets から距離を広げる。空いたマスは重み1、プレイヤー側でないキャラクターの
// マスは重み gc.DistanceMapObstacleCost、壁や地形は通れない。置物とプレイヤー側のマスは rule で決める
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
		if rule.blockDeployed && world.Components.Deployed.Has(e) {
			m.Cost[m.Index(g.Coord)] = 0
			continue
		}
		if IsAlly(world, e) && !rule.avoidAllies {
			continue
		}
		m.Cost[m.Index(g.Coord)] = gc.DistanceMapObstacleCost
	}
	m.Spread(targets)
}
