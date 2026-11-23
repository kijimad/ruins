package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkTypeAt_開始チャンクは墜落現場で区域が重ならない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	_, _, cols, rows := dungeon.DungeonOverworld.BandShape()
	spawnRow := gc.SeamlessBand{Rows: rows}.SpawnChunkY()
	start := spawnChunkOf(cols, spawnRow)
	for seed := range uint64(100) {
		assert.Equal(t, chunkCrashSite, chunkTypeAt(raws, seed, start, cols, spawnRow), "seed=%d", seed)
		for y := spawnRow - 20; y <= spawnRow+20; y++ {
			for x := range cols {
				r, ok := districtAt(raws, seed, consts.Coord[consts.Chunk]{X: x, Y: y}, cols, spawnRow)
				if ok {
					assert.False(t, r.contains(start), "seed=%d 区域 %v は開始チャンクを含まない", seed, r.anchor)
				}
			}
		}
	}
}

func TestBuildRoadOverlay_道の経路が墜落現場を通っても墜落現場へ道をつながない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	const cols, spawnRow consts.Chunk = 7, 4
	start := spawnChunkOf(cols, spawnRow)
	area := MacroRange{OriginX: 0, OriginY: spawnRow - 20, Cols: cols, Rows: 41}
	crossed := 0
	for seed := range uint64(200) {
		if !onRoad(raws, seed, start, cols) {
			continue
		}
		crossed++
		overlay := buildRoadOverlay(raws, seed, area, cols, spawnRow)
		assert.Zero(t, overlay[start], "seed=%d", seed)
		for _, n := range []struct {
			c   consts.Coord[consts.Chunk]
			dir RoadDir
		}{
			{consts.Coord[consts.Chunk]{X: start.X, Y: start.Y - 1}, RoadS},
			{consts.Coord[consts.Chunk]{X: start.X, Y: start.Y + 1}, RoadN},
			{consts.Coord[consts.Chunk]{X: start.X - 1, Y: start.Y}, RoadE},
			{consts.Coord[consts.Chunk]{X: start.X + 1, Y: start.Y}, RoadW},
		} {
			assert.Zero(t, overlay[n.c]&n.dir, "seed=%d %v", seed, n.c)
		}
	}
	require.Positive(t, crossed, "前提: 道の経路が開始チャンクを通る seed がある")
}

func TestCrashSites_全ての絵は奇数の大きさでチャンクに収まる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	chunkW, chunkH, _, _ := dungeon.DungeonOverworld.BandShape()
	for _, site := range raw.PtrSlice(raws.CrashSites) {
		names := loader.FacilityChunkNames(site.Id)
		require.NotEmpty(t, names, site.Id)
		for _, name := range names {
			_, _, cells, err := loader.LoadTemplateByName(name, 1)
			require.NoError(t, err, name)
			h, wd := len(cells), len(cells[0])
			assert.Equal(t, 1, h%2, "%s の高さは真ん中のマスを持つ奇数", name)
			assert.Equal(t, 1, wd%2, "%s の幅は真ん中のマスを持つ奇数", name)
			assert.LessOrEqual(t, wd, int(chunkW), name)
			assert.LessOrEqual(t, h, int(chunkH), name)
		}
	}
}

// crashSiteGroundNames は墜落現場の地面のタイルに載る、タイル以外の名前を集める
func crashSiteGroundNames(world w.World) []string {
	ground := map[consts.Coord[consts.Tile]]bool{}
	tiles := ecs.NewFilter2[gc.GridElement, gc.Tile](world.ECS).Query()
	for tiles.Next() {
		e := tiles.Entity()
		if n := world.Components.Name.Get(e).Name; n == "Impact Crater" || n == "Scorched Ground" {
			ground[world.Components.GridElement.Get(e).Coord] = true
		}
	}
	var names []string
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Without(ecs.C[gc.Tile]()).Query()
	for q.Next() {
		e := q.Entity()
		if ground[world.Components.GridElement.Get(e).Coord] {
			names = append(names, string(world.Components.Name.Get(e).Name))
		}
	}
	return names
}

// cargoPodContents は貨物ポッドに入っているアイテムの名前を集める
func cargoPodContents(world w.World) []string {
	var names []string
	q := ecs.NewFilter2[gc.LocationInStorage, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		owner := world.Components.LocationInStorage.Get(e).Owner
		if world.Components.Name.Has(owner) && world.Components.Name.Get(owner).Name == "Cargo Pod" {
			names = append(names, string(world.Components.Name.Get(e).Name))
		}
	}
	return names
}

func TestDriver_Start_墜落現場で始まる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	const cols, rows consts.Chunk = 3, 3
	for _, seed := range []uint64{1, 777, 12345} {
		world := testutil.InitTestWorld(t)
		s := NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, chunkW, chunkH, cols, rows), &NewGameParams{RunSeed: seed})
		require.NoError(t, s.Start(world))

		start := spawnChunkOf(cols, query.GetSeamlessBand(world).SpawnChunkY())
		center := consts.Coord[consts.Tile]{X: start.X.Tiles(chunkW) + chunkW/2, Y: start.Y.Tiles(chunkH) + chunkH/2}
		player, err := query.GetPlayerEntity(world)
		require.NoError(t, err)
		assert.Equal(t, center, world.Components.GridElement.Get(player).Coord, "seed=%d プレイヤーは絵の真ん中に立つ", seed)

		playerName := string(world.Components.Name.Get(player).Name)
		assert.ElementsMatch(t,
			[]string{playerName, "Cube", "Burning Wreckage", "Wreckage", "Wreckage", "Wreckage", "Cargo Pod"},
			crashSiteGroundNames(world),
			"seed=%d 墜落現場の地面には絵の置物とプレイヤーとキューブだけが載り、散布物も戦利品も敵も載らない", seed)
		assert.ElementsMatch(t, []string{"Bandage", "Ration Bar", "Camp Bed", "Iron Knife"}, cargoPodContents(world), "seed=%d", seed)
	}
}
