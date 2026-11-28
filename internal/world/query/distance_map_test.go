package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDistanceMap(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}

	t.Run("HP を持つ通れない置物は重く、壁は通れない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 6, 1
		si.Blocked = map[gc.GridElement]bool{{Coord: at(2, 0)}: true, {Coord: at(4, 0)}: true}
		si.Characters = map[gc.GridElement]ecs.Entity{}
		door := world.ECS.NewEntity()
		world.Components.Prop.Add(door, &gc.Prop{})
		world.Components.HP.Add(door, &gc.HP{})
		world.Components.GridElement.Add(door, &gc.GridElement{Coord: at(2, 0)})

		m := gc.NewDistanceMap()
		buildDistanceMap(world, m, []consts.Coord[consts.Tile]{at(0, 0)}, baseRule)
		d, ok := m.Dist(at(3, 0))
		require.True(t, ok)
		assert.Equal(t, int32(1+gc.DistanceMapObstacleCost+1), d, "扉を壊して抜ける")
		_, ok = m.Dist(at(5, 0))
		assert.False(t, ok, "壁の向こうは届かない")
	})

	t.Run("味方のマスは軽く、ほかのキャラクターのマスは重い", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		ally := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(ally, &gc.FactionAlly{})
		beast := world.ECS.NewEntity()
		world.Components.FactionNeutral.Add(beast, &gc.FactionNeutral{})
		si := GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 3, 2
		si.Blocked = map[gc.GridElement]bool{}
		si.Characters = map[gc.GridElement]ecs.Entity{{Coord: at(1, 0)}: ally, {Coord: at(1, 1)}: beast}

		m := gc.NewDistanceMap()
		buildDistanceMap(world, m, []consts.Coord[consts.Tile]{at(0, 0)}, baseRule)
		dAlly, _ := m.Dist(at(1, 0))
		dBeast, _ := m.Dist(at(1, 1))
		assert.Equal(t, []int32{1, gc.DistanceMapObstacleCost}, []int32{dAlly, dBeast})
	})
}

func TestGetBaseDistanceMap_引いたときに作り無効化で捨てる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	m := GetSingleton[gc.BaseDistanceMap](world, world.Components.BaseDistanceMap)
	require.False(t, m.Built, "前提: 作られる前")

	assert.Same(t, &m.DistanceMap, GetBaseDistanceMap(world))
	assert.True(t, m.Built, "引いたときに作る")
	InvalidateDistanceMaps(world)
	assert.False(t, m.Built, "無効化で捨てる")
}

// BenchmarkBuildDistanceMap は15×15チャンクの帯の全体へ、距離の地図を1回作る時間を測る。襲撃の敵がいる手番ごとに作る
func BenchmarkBuildDistanceMap(b *testing.B) {
	world := testutil.InitTestWorld(b)
	si := GetSpatialIndex(world)
	const size = 15 * 24
	si.Built, si.MapWidth, si.MapHeight = true, size, size
	si.Blocked = map[gc.GridElement]bool{}
	si.Characters = map[gc.GridElement]ecs.Entity{}
	for y := 0; y < size; y += 6 {
		for x := range size - 3 {
			si.Blocked[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x + y%12/6*3), Y: consts.Tile(y)}}] = true
		}
	}
	targets := []consts.Coord[consts.Tile]{{X: size / 2, Y: size / 2}}
	m := gc.NewDistanceMap()
	buildDistanceMap(world, m, targets, baseRule)
	b.ResetTimer()
	for b.Loop() {
		buildDistanceMap(world, m, targets, baseRule)
	}
}

func TestGetPlayerDistanceMap(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}

	t.Run("プレイヤーを的にし、HPを持つ置物は通れず、味方のマスは重い", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: at(0, 0)})
		ally := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(ally, &gc.FactionAlly{})
		si := GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 5, 1
		si.Blocked = map[gc.GridElement]bool{{Coord: at(3, 0)}: true}
		si.Characters = map[gc.GridElement]ecs.Entity{{Coord: at(1, 0)}: ally}
		door := world.ECS.NewEntity()
		world.Components.Prop.Add(door, &gc.Prop{})
		world.Components.HP.Add(door, &gc.HP{})
		world.Components.GridElement.Add(door, &gc.GridElement{Coord: at(3, 0)})

		m := GetPlayerDistanceMap(world)

		dists := make([]int32, 5)
		for x := range dists {
			d, ok := m.Dist(at(x, 0))
			if !ok {
				d = -1
			}
			dists[x] = d
		}
		assert.Equal(t, []int32{0, gc.DistanceMapObstacleCost, gc.DistanceMapObstacleCost + 1, -1, -1}, dists, "味方のマスは重く、扉を壊さないので向こう側へは届かない")
	})

	t.Run("敵の手番の初めに無効化すると次に引いたときに作り直す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		GetPlayerDistanceMap(world)
		GetBaseDistanceMap(world)

		InvalidateDistanceMaps(world)

		follow := GetSingleton[gc.PlayerDistanceMap](world, world.Components.PlayerDistanceMap)
		raid := GetSingleton[gc.BaseDistanceMap](world, world.Components.BaseDistanceMap)
		assert.Equal(t, []bool{false, false}, []bool{follow.Built, raid.Built})
	})
}
