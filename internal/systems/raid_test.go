package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// firstWaveTurn は7日目の最初の波の経過ターン
const firstWaveTurn = 6*gc.TurnsPerDay + 1000

func TestRaidWaveSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		total int
		want  []int
	}{
		{"割り切れる", 6, []int{2, 2, 2}},
		{"余りは後の波に回す", 8, []int{2, 3, 3}},
		{"余り1は最後の波", 10, []int{3, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := make([]int, 0, query.RaidWaveCount)
			for i := range query.RaidWaveCount {
				got = append(got, raidWaveSize(tt.total, i))
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRaidNightSize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 6, raidNightSize(1))
	assert.Equal(t, raidMaxPerNight, raidNightSize(41), "北の奥でも上限を超えない")
}

func TestRaidSystem_波のターンに展開中のキューブを囲んで印のある敵を湧かせ寝ているプレイヤーを起こす(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	center := world.Components.GridElement.Get(cube).Coord
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.Activity.Add(player, activity.NewSleepActivity())
	world.Components.Sleeping.Add(player, &gc.Sleeping{})
	query.GetGameTime(world).TotalTurns = firstWaveTurn

	require.NoError(t, (&RaidSystem{}).Update(world))

	raiders := raiderTiles(world)
	assert.Len(t, raiders, raidWaveSize(raidNightSize(raidDanger(world, center)), 0))
	for _, c := range raiders {
		d := geometry.ChebyshevDistance(center, c)
		assert.True(t, d >= int(raidSpawnMin) && d <= int(raidSpawnMax), "輪の上に湧く: %v は %d 離れている", c, d)
	}
	assert.False(t, world.Components.Sleeping.Has(player), "波で目を覚ます")
}

func TestRaidSystem_視界の中には湧かせず輪を外へ広げる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	center := world.Components.GridElement.Get(cube).Coord
	const seen = 30
	visible := query.GetVisionState(world).VisibleTiles
	for y := center.Y - seen; y <= center.Y+seen; y++ {
		for x := center.X - seen; x <= center.X+seen; x++ {
			visible[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}] = true
		}
	}
	query.GetGameTime(world).TotalTurns = firstWaveTurn

	require.NoError(t, (&RaidSystem{}).Update(world))

	raiders := raiderTiles(world)
	require.NotEmpty(t, raiders)
	for _, c := range raiders {
		assert.Greater(t, geometry.ChebyshevDistance(center, c), seen, "見えている所には湧かない: %v", c)
	}
}

func TestRaidSystem_夜明けに印を外して待機に戻し敵は残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	query.GetGameTime(world).TotalTurns = firstWaveTurn
	require.NoError(t, (&RaidSystem{}).Update(world))
	var spawned []ecs.Entity
	q := ecs.NewFilter1[gc.Raider](world.ECS).Query()
	for q.Next() {
		spawned = append(spawned, q.Entity())
	}
	require.NotEmpty(t, spawned)
	for _, e := range spawned {
		world.Components.SoloAI.Get(e).SubState = gc.AIStateChasing
	}

	query.GetGameTime(world).TotalTurns = 7 * gc.TurnsPerDay
	require.NoError(t, (&RaidSystem{}).Update(world))

	for _, e := range spawned {
		assert.True(t, world.ECS.Alive(e), "引き上げずに残る")
		assert.False(t, world.Components.Raider.Has(e), "印が外れる")
		assert.Equal(t, gc.AIStateWaiting, world.Components.SoloAI.Get(e).SubState, "待機に戻る")
	}
}

func TestRunTurnEndSystems_登録したRaidSystemが波を湧かせる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	query.GetGameTime(world).TotalTurns = firstWaveTurn
	updaters, _ := InitializeSystems(world)
	name := (&RaidSystem{}).String()
	sys, ok := updaters[name].(*RaidSystem)
	require.True(t, ok, "InitializeSystems に登録する")
	world.Updaters = map[string]w.Updater{name: sys}

	require.NoError(t, runTurnEndSystems(world))
	assert.NotEmpty(t, raiderTiles(world))
}

func raiderTiles(world w.World) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	q := ecs.NewFilter2[gc.Raider, gc.GridElement](world.ECS).Query()
	for q.Next() {
		_, g := q.Get()
		out = append(out, g.Coord)
	}
	return out
}
