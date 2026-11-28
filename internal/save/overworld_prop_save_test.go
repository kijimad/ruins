package save

import (
	"fmt"
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWorld_地上に置いた置き物は読み込んで地上へ戻っても同じマスに残る(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	definition := dungeon.NewOverworldDefinition("o", 0, 24, 24, 5, 5)

	world := testutil.InitTestWorld(t)
	require.NoError(t, overworld.NewDriver(mapplanner.PlannerTypeOverworldField, definition, &overworld.NewGameParams{RunSeed: 42}).Start(world))
	cube, ok := query.ActiveCube(world)
	require.True(t, ok)
	// 開始地点は墜落現場の置き物が野営の範囲に入り DeployCube できないので、展開の印だけ付けて空いたマスに置く。
	// 印は読み込んだ後に帯を止め、置いた行が流れて消えないようにする役も兼ねる
	r := query.CubeDeployRange(world, cube)
	world.Components.Deployed.Add(cube, &gc.Deployed{Range: r})
	base := world.Components.GridElement.Get(cube).Coord
	var placedAt consts.Coord[consts.Tile]
	found := false
	for dy := -r.Y; dy <= r.Y && !found; dy++ {
		for dx := -r.X; dx <= r.X && !found; dx++ {
			c := base.Add(consts.Coord[consts.Tile]{X: dx, Y: dy})
			if (dx != 0 || dy != 0) && lifecycle.FacilityCellFree(world, cube, c) {
				placedAt, found = c, true
			}
		}
	}
	require.True(t, found, "前提: 野営に空いたマスがある")
	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	_, err = lifecycle.PlaceFacility(world, cube, item, placedAt)
	require.NoError(t, err)
	before := fieldProps(world)
	require.Contains(t, before, fmt.Sprintf("%s@%v", query.GetEntityName(item, world), placedAt), "前提: 置いた設備が地上にある")
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))
	require.NoError(t, overworld.NewDriver(mapplanner.PlannerTypeOverworldField, definition, nil).Start(fresh))

	assert.Equal(t, before, fieldProps(fresh), "地上へ戻っても作り直さず、置き物が増えも減りもしない")
	var deployable []bool
	for _, e := range query.GetEntitiesAt(fresh, placedAt.X, placedAt.Y) {
		if fresh.Components.Prop.Has(e) {
			deployable = append(deployable, fresh.Components.Deployable.Has(e))
		}
	}
	assert.Equal(t, []bool{true}, deployable, "置いた設備は設備のまま戻り、設備画面から外せる")
}

// fieldProps は現ステージの地上にある置き物を「名前@位置」で並べて返す
func fieldProps(world w.World) []string {
	var out []string
	q := query.ActiveFilter3[gc.Prop, gc.GridElement, gc.LocationOnField](world).Query()
	for q.Next() {
		_, g, _ := q.Get()
		out = append(out, fmt.Sprintf("%s@%v", query.GetEntityName(q.Entity(), world), g.Coord))
	}
	slices.Sort(out)
	return out
}
