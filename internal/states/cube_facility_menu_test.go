package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deployedFacilityState は展開済みキューブと設備画面ステートを用意する。キューブは(10,10)、範囲は基準の2。
func deployedFacilityState(t *testing.T) (w.World, *CubeFacilityMenuState, ecs.Entity) {
	t.Helper()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, lifecycle.DeployCube(world, cube))
	return world, &CubeFacilityMenuState{cube: cube}, cube
}

func TestCubeFacilityMenu_moveCursorは範囲でクランプする(t *testing.T) {
	t.Parallel()
	world, st, _ := deployedFacilityState(t)

	st.moveCursor(world, 1, 0)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 1, Y: 0}, st.cursor)
	st.moveCursor(world, 1, 0)
	st.moveCursor(world, 1, 0)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 2, Y: 0}, st.cursor, "範囲外へは出ない")
	for range 5 {
		st.moveCursor(world, -1, 0)
	}
	assert.Equal(t, consts.Coord[consts.Tile]{X: -2, Y: 0}, st.cursor)
}

func TestCubeFacilityMenu_cellKindAtがマス種別を分ける(t *testing.T) {
	t.Parallel()
	world, st, cube := deployedFacilityState(t)
	base := world.Components.GridElement.Get(cube).Coord

	assert.Equal(t, hud.FacilityCellCube, st.cellKindAt(world, base))

	empty := base.Add(consts.Coord[consts.Tile]{X: 1, Y: 0})
	assert.Equal(t, hud.FacilityCellEmpty, st.cellKindAt(world, empty))

	blocked := base.Add(consts.Coord[consts.Tile]{X: 0, Y: 1})
	wall := world.ECS.NewEntity()
	world.Components.GridElement.Add(wall, &gc.GridElement{Coord: blocked})
	world.Components.BlockPass.Add(wall, &gc.BlockPass{})
	query.InvalidateSpatialIndex(world)
	assert.Equal(t, hud.FacilityCellBlocked, st.cellKindAt(world, blocked))

	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	used := base.Add(consts.Coord[consts.Tile]{X: -1, Y: 0})
	_, err = lifecycle.PlaceFacility(world, cube, item, used)
	require.NoError(t, err)
	assert.Equal(t, hud.FacilityCellUsed, st.cellKindAt(world, used))
}

func TestCubeFacilityMenu_cursorInfoが内容と操作を返す(t *testing.T) {
	t.Parallel()
	world, st, cube := deployedFacilityState(t)
	base := world.Components.GridElement.Get(cube).Coord

	content, hint := st.cursorInfo(world, base)
	assert.Equal(t, query.T(world, "Cube"), content)
	assert.Empty(t, hint, "キューブ本体に操作は無い")

	content, hint = st.cursorInfo(world, base.Add(consts.Coord[consts.Tile]{X: 1, Y: 0}))
	assert.Equal(t, query.T(world, "Empty"), content)
	assert.Equal(t, query.T(world, "Enter: place"), hint)
}

func TestCubeFacilityMenu_selectCellの空きマスは据付選択へ進む(t *testing.T) {
	t.Parallel()
	world, st, _ := deployedFacilityState(t)
	// 据える候補をバックパックに用意する。無いとログのみで進まない
	_, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	st.cursor = consts.Coord[consts.Tile]{X: 1, Y: 0}

	trans, err := st.selectCell(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransPush, trans.Type, "空きマスで決定すると据付アイテム選択へ進む")
}

func TestCubeFacilityMenu_selectCellの設備マスは撤去する(t *testing.T) {
	t.Parallel()
	world, st, cube := deployedFacilityState(t)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	coord := world.Components.GridElement.Get(cube).Add(consts.Coord[consts.Tile]{X: 1, Y: 0})
	_, err = lifecycle.PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)
	st.cursor = consts.Coord[consts.Tile]{X: 1, Y: 0}

	_, err = st.selectCell(world)
	require.NoError(t, err)
	assert.True(t, world.Components.LocationInBackpack.Has(item), "設備マスで決定すると撤去してバックパックへ戻す")
	assert.Equal(t, player, world.Components.LocationInBackpack.Get(item).Owner)
}
