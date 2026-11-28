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
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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
	assert.Equal(t, 15, raidNightSize(1))
	assert.Equal(t, raidMaxPerNight, raidNightSize(41), "北の奥でも上限を超えない")
}

func TestRaidSystem_波のターンに展開中のキューブを囲んで襲撃の敵を湧かせ寝ているプレイヤーを起こす(t *testing.T) {
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
	assert.Contains(t, logTexts(world), "They are coming out of the dark.", "波が来たことをログで知らせる")
}

func TestRaidSystem_寝ていなければ行動を止めない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.Activity.Add(player, activity.NewActivity(gc.BehaviorRest, 10))
	query.GetGameTime(world).TotalTurns = firstWaveTurn

	require.NoError(t, (&RaidSystem{}).Update(world))

	require.NotEmpty(t, raiderTiles(world))
	assert.True(t, world.Components.Activity.Has(player), "眠り以外の行動は続く")
}

func TestRaidSystem_キューブが無ければプレイヤーを囲んで湧かせる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	world.ECS.RemoveEntity(cube)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	center := world.Components.GridElement.Get(player).Coord
	query.GetGameTime(world).TotalTurns = firstWaveTurn

	require.NoError(t, (&RaidSystem{}).Update(world))

	raiders := raiderTiles(world)
	require.NotEmpty(t, raiders)
	for _, c := range raiders {
		d := geometry.ChebyshevDistance(center, c)
		assert.True(t, d >= int(raidSpawnMin) && d <= int(raidSpawnMax), "プレイヤーを囲む輪に湧く: %v は %d 離れている", c, d)
	}
}

func TestRaidSystem_畳んだキューブも囲んで湧かせる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	require.False(t, world.Components.Deployed.Has(cube), "前提: 開始時のキューブは畳んである")
	center := world.Components.GridElement.Get(cube).Coord
	query.GetGameTime(world).TotalTurns = firstWaveTurn

	require.NoError(t, (&RaidSystem{}).Update(world))

	raiders := raiderTiles(world)
	require.NotEmpty(t, raiders)
	for _, c := range raiders {
		d := geometry.ChebyshevDistance(center, c)
		assert.True(t, d >= int(raidSpawnMin) && d <= int(raidSpawnMax), "畳んだキューブを囲む輪に湧く: %v は %d 離れている", c, d)
	}
}

func TestRaidSystem_地上にいなければ予告だけする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, "ash")
	require.NoError(t, err)

	query.GetGameTime(world).TotalTurns = 6 * gc.TurnsPerDay
	require.NoError(t, (&RaidSystem{}).Update(world))
	assert.Equal(t, []string{"Something is gathering in the snow. They will come tonight."}, logTexts(world), "予告は地上の外でも出す")

	raider, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 30, Y: 30}, "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge

	query.GetGameTime(world).TotalTurns = firstWaveTurn
	require.NoError(t, (&RaidSystem{}).Update(world))
	assert.Equal(t, []consts.Coord[consts.Tile]{{X: 30, Y: 30}}, raiderTiles(world), "地上の外では波を湧かせない")
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

func TestRaidSystem_キューブから離れたプレイヤーのそばには湧かせない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	center := world.Components.GridElement.Get(cube).Coord
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	playerAt := center.Add(consts.Coord[consts.Tile]{X: 20})
	world.Components.GridElement.Get(player).Coord = playerAt
	query.InvalidateSpatialIndex(world)
	// 最初の輪のうちプレイヤーから離れたマスを全て見えている扱いにし、残りをプレイヤーの近くのマスだけにする
	visible := query.GetVisionState(world).VisibleTiles
	for y := center.Y - raidSpawnMax; y <= center.Y+raidSpawnMax; y++ {
		for x := center.X - raidSpawnMax; x <= center.X+raidSpawnMax; x++ {
			c := consts.Coord[consts.Tile]{X: x, Y: y}
			if geometry.ChebyshevDistance(c, playerAt) >= raidPlayerMin {
				visible[gc.GridElement{Coord: c}] = true
			}
		}
	}
	query.GetGameTime(world).TotalTurns = firstWaveTurn

	require.NoError(t, (&RaidSystem{}).Update(world))

	raiders := raiderTiles(world)
	require.NotEmpty(t, raiders)
	for _, c := range raiders {
		assert.GreaterOrEqual(t, geometry.ChebyshevDistance(playerAt, c), raidPlayerMin, "プレイヤーの近くには湧かない: %v", c)
	}
}

func TestRaidSystem_夜が明けても湧いた敵は襲撃の動きのまま残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	query.GetGameTime(world).TotalTurns = firstWaveTurn
	require.NoError(t, (&RaidSystem{}).Update(world))
	spawned := raiderTiles(world)
	require.NotEmpty(t, spawned)

	query.GetGameTime(world).TotalTurns = 7 * gc.TurnsPerDay
	require.NoError(t, (&RaidSystem{}).Update(world))

	assert.Equal(t, spawned, raiderTiles(world))
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
	q := ecs.NewFilter2[gc.SoloAI, gc.GridElement](world.ECS).Query()
	for q.Next() {
		solo, g := q.Get()
		if solo.Movement == rawschema.Converge {
			out = append(out, g.Coord)
		}
	}
	return out
}

func logTexts(world w.World) []string {
	entries := query.GetGameLog(world).GetRecentEntries(20)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Text())
	}
	return out
}
