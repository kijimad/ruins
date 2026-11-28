package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRaidMap(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}

	t.Run("空いた通路は1マスごとに1遠くなる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 4, 1
		si.Blocked = map[gc.GridElement]bool{}
		si.Characters = map[gc.GridElement]ecs.Entity{}

		m := BuildRaidMap(world, []consts.Coord[consts.Tile]{at(0, 0)}, nil)
		got := make([]int32, 0, 4)
		for x := range 4 {
			d, ok := m.Dist(at(x, 0))
			require.True(t, ok)
			got = append(got, d)
		}
		assert.Equal(t, []int32{0, 1, 2, 3}, got)
	})

	t.Run("HP を持つ通れない置物は重く、壁は通れない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 6, 1
		si.Blocked = map[gc.GridElement]bool{{Coord: at(2, 0)}: true, {Coord: at(4, 0)}: true}
		si.Characters = map[gc.GridElement]ecs.Entity{}
		door := world.ECS.NewEntity()
		world.Components.Prop.Add(door, &gc.Prop{})
		world.Components.HP.Add(door, &gc.HP{})
		world.Components.GridElement.Add(door, &gc.GridElement{Coord: at(2, 0)})

		m := BuildRaidMap(world, []consts.Coord[consts.Tile]{at(0, 0)}, nil)
		d, ok := m.Dist(at(3, 0))
		require.True(t, ok)
		assert.Equal(t, int32(1+raidObstacleCost+1), d, "扉を壊して抜ける")
		_, ok = m.Dist(at(5, 0))
		assert.False(t, ok, "壁の向こうは届かない")
	})

	t.Run("プレイヤーのマスは軽く、ほかのキャラクターのマスは重い", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
		beast := world.ECS.NewEntity()
		world.Components.FactionNeutral.Add(beast, &gc.FactionNeutral{})
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 3, 2
		si.Blocked = map[gc.GridElement]bool{}
		si.Characters = map[gc.GridElement]ecs.Entity{{Coord: at(1, 0)}: player, {Coord: at(1, 1)}: beast}

		m := BuildRaidMap(world, []consts.Coord[consts.Tile]{at(0, 0)}, nil)
		dPlayer, _ := m.Dist(at(1, 0))
		dBeast, _ := m.Dist(at(1, 1))
		assert.Equal(t, int32(1), dPlayer)
		assert.Equal(t, int32(raidObstacleCost), dBeast)
	})

	t.Run("縦横の隣がどちらも塞がった角は斜めに抜けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 2, 2
		si.Blocked = map[gc.GridElement]bool{{Coord: at(1, 0)}: true, {Coord: at(0, 1)}: true}
		si.Characters = map[gc.GridElement]ecs.Entity{}

		m := BuildRaidMap(world, []consts.Coord[consts.Tile]{at(0, 0)}, nil)
		_, ok := m.Dist(at(1, 1))
		assert.False(t, ok)
	})

	t.Run("Steps は角の向こうの的へ斜めに抜けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 3, 3
		si.Blocked = map[gc.GridElement]bool{{Coord: at(1, 0)}: true, {Coord: at(0, 1)}: true}
		si.Characters = map[gc.GridElement]ecs.Entity{}

		m := BuildRaidMap(world, []consts.Coord[consts.Tile]{at(0, 0), at(2, 2)}, nil)
		assert.Equal(t, []consts.Coord[consts.Tile]{at(2, 2)}, m.Steps(world, at(1, 1), nil), "角の向こうの (0,0) でなく、斜めに抜けられる (2,2) へ進む")
	})

	t.Run("的が無ければどのマスにも届かない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 3, 1
		si.Blocked = map[gc.GridElement]bool{}
		si.Characters = map[gc.GridElement]ecs.Entity{}

		m := BuildRaidMap(world, nil, nil)
		_, ok := m.Dist(at(1, 0))
		assert.False(t, ok)
	})

	t.Run("Steps は下る隣を距離の小さい順に返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built, si.MapWidth, si.MapHeight = true, 5, 5
		si.Blocked = map[gc.GridElement]bool{}
		si.Characters = map[gc.GridElement]ecs.Entity{}

		m := BuildRaidMap(world, []consts.Coord[consts.Tile]{at(0, 0)}, nil)
		assert.Equal(t, []consts.Coord[consts.Tile]{at(2, 1), at(2, 0), at(2, 2)}, m.Steps(world, at(3, 1), nil))
		assert.Empty(t, m.Steps(world, at(0, 0), nil), "的の上では進む先が無い")
	})
}

// BenchmarkBuildRaidMap は15×15チャンクの帯の全体へ、距離の地図を1回広げる時間を測る。襲撃の夜は敵の手番ごとに作る
func BenchmarkBuildRaidMap(b *testing.B) {
	world := testutil.InitTestWorld(b)
	si := query.GetSpatialIndex(world)
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
	m := BuildRaidMap(world, targets, nil)
	b.ResetTimer()
	for b.Loop() {
		m = BuildRaidMap(world, targets, m)
	}
}
