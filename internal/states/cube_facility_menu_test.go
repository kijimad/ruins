package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
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

	kind, _ := st.cellKindAt(world, base)
	assert.Equal(t, hud.FacilityCellCube, kind)

	empty := base.Add(consts.Coord[consts.Tile]{X: 1, Y: 0})
	kind, _ = st.cellKindAt(world, empty)
	assert.Equal(t, hud.FacilityCellEmpty, kind)

	blocked := base.Add(consts.Coord[consts.Tile]{X: 0, Y: 1})
	wall := world.ECS.NewEntity()
	world.Components.GridElement.Add(wall, &gc.GridElement{Coord: blocked})
	world.Components.BlockPass.Add(wall, &gc.BlockPass{})
	query.InvalidateSpatialIndex(world)
	kind, _ = st.cellKindAt(world, blocked)
	assert.Equal(t, hud.FacilityCellBlocked, kind)

	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	used := base.Add(consts.Coord[consts.Tile]{X: -1, Y: 0})
	_, err = lifecycle.PlaceFacility(world, cube, item, used)
	require.NoError(t, err)
	kind, facility := st.cellKindAt(world, used)
	assert.Equal(t, hud.FacilityCellUsed, kind)
	assert.Equal(t, item, facility, "Used のとき装着されている設備を併せて返す")
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

	// 装着した設備のマスは設備名と撤去操作を出す
	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	used := base.Add(consts.Coord[consts.Tile]{X: -1, Y: 0})
	_, err = lifecycle.PlaceFacility(world, cube, item, used)
	require.NoError(t, err)
	content, hint = st.cursorInfo(world, used)
	assert.Equal(t, query.GetEntityName(item, world), content)
	assert.Equal(t, query.T(world, "Enter: remove"), hint)
}

func TestCubeFacilityMenu_doActionが入力を捌く(t *testing.T) {
	t.Parallel()
	world, st, _ := deployedFacilityState(t)

	trans, err := st.doAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, trans.Type, "キャンセルで閉じる")

	st.cursor = consts.Coord[consts.Tile]{X: 0, Y: 0}
	_, err = st.doAction(world, inputmapper.ActionMenuRight)
	require.NoError(t, err)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 1, Y: 0}, st.cursor, "方向キーでカーソルが動く")
}

func TestCubeFacilityMenu_selectCellの空きマスは装着選択へ進む(t *testing.T) {
	t.Parallel()
	world, st, _ := deployedFacilityState(t)
	// 装着する候補をバックパックに用意する。無いとログのみで進まない
	_, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	st.cursor = consts.Coord[consts.Tile]{X: 1, Y: 0}

	trans, err := st.selectCell(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransPush, trans.Type, "空きマスで決定すると装着アイテム選択へ進む")
}

func TestPlaceFacilityChoice_範囲内は装着し範囲外は何もしない(t *testing.T) {
	t.Parallel()
	world, _, cube := deployedFacilityState(t)
	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	coord := world.Components.GridElement.Get(cube).Add(consts.Coord[consts.Tile]{X: 1, Y: 0})

	// 範囲外は装着しない
	require.NoError(t, placeFacilityChoice(world, cube, coord, []ecs.Entity{item}, 5))
	assert.True(t, world.Components.LocationInBackpack.Has(item), "範囲外の idx では装着しない")

	// 範囲内は装着する
	require.NoError(t, placeFacilityChoice(world, cube, coord, []ecs.Entity{item}, 0))
	assert.True(t, world.Components.LocationOnField.Has(item), "範囲内の idx で装着する")
}

func TestCubeFacilityMenu_Updateは展開中でなければ閉じる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	st := &CubeFacilityMenuState{cube: cube}

	trans, err := st.Update(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, trans.Type, "展開中でなければ設備画面は開けず閉じる")

	// 遷移フレームで Draw が先に来ても Deployed 不在で panic しない
	require.NoError(t, st.Draw(world, nil), "展開中でなければ描かず nil を返す")
}

func TestCubeFacilitySelect_Fetchが装着候補を返す(t *testing.T) {
	t.Parallel()
	world, _, cube := deployedFacilityState(t)
	_, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)

	st := &CubeFacilitySelectState{cube: cube, coord: consts.Coord[consts.Tile]{X: 11, Y: 10}}
	props, err := st.Fetch(world)
	require.NoError(t, err)
	assert.Len(t, props.Candidates, 1, "バックパックの装着アイテムを候補に出す")
}

func TestCubeFacilitySelect_Fetchはプレイヤー不在でエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	st := &CubeFacilitySelectState{cube: world.ECS.NewEntity()}
	_, err := st.Fetch(world)
	require.Error(t, err, "プレイヤー不在は握りつぶさず error")
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
