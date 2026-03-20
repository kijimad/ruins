package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusState_OnStart(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.mount, "mountが初期化されている")
}

func TestStatusState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	assert.Equal(t, 5, len(props.Tabs), "タブは5つ（基本、能力、スキル、効果、健康）")
	assert.Equal(t, "basic", props.Tabs[0].ID)
	assert.Equal(t, "attributes", props.Tabs[1].ID)
	assert.Equal(t, "skills", props.Tabs[2].ID)
	assert.Equal(t, "effects", props.Tabs[3].ID)
	assert.Equal(t, "health", props.Tabs[4].ID)
}

func TestStatusState_TabNavigation(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.mount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(state.mount.Store(), "status", hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	state.mount.Update()

	// 初期状態
	tabIndex, _ := hooks.GetState[int](state.mount, "status_tabIndex")
	assert.Equal(t, 0, tabIndex, "初期タブインデックスは0")

	// 右に移動
	state.mount.Dispatch(inputmapper.ActionMenuTabNext)
	tabIndex, _ = hooks.GetState[int](state.mount, "status_tabIndex")
	assert.Equal(t, 1, tabIndex, "右移動後は1")

	// 最後のタブまで移動
	for i := 2; i < len(props.Tabs); i++ {
		state.mount.Dispatch(inputmapper.ActionMenuTabNext)
	}
	tabIndex, _ = hooks.GetState[int](state.mount, "status_tabIndex")
	assert.Equal(t, len(props.Tabs)-1, tabIndex, "最後のタブ")

	// 循環して最初に戻る
	state.mount.Dispatch(inputmapper.ActionMenuTabNext)
	hooks.UseTabMenu(state.mount.Store(), "status", hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	tabIndex, _ = hooks.GetState[int](state.mount, "status_tabIndex")
	assert.Equal(t, 0, tabIndex, "循環して最初に戻る")
}

func TestStatusState_ItemNavigation(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.mount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(state.mount.Store(), "status", hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	state.mount.Update()

	// 初期状態
	itemIndex, _ := hooks.GetState[int](state.mount, "status_itemIndex")
	assert.Equal(t, 0, itemIndex, "初期アイテムインデックスは0")

	// 下に移動
	state.mount.Dispatch(inputmapper.ActionMenuDown)
	itemIndex, _ = hooks.GetState[int](state.mount, "status_itemIndex")
	assert.Equal(t, 1, itemIndex, "下移動後は1")

	// 上に移動
	state.mount.Dispatch(inputmapper.ActionMenuUp)
	itemIndex, _ = hooks.GetState[int](state.mount, "status_itemIndex")
	assert.Equal(t, 0, itemIndex, "上移動後は0")
}

func TestStatusState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestStatusState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "CloseMenuでTransPop")
}

func TestStatusState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ナビゲーションアクションはTransNoneを返す
	actions := []inputmapper.ActionID{
		inputmapper.ActionMenuUp,
		inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft,
		inputmapper.ActionMenuRight,
		inputmapper.ActionMenuTabNext,
		inputmapper.ActionMenuTabPrev,
	}

	for _, action := range actions {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
}

func TestStatusState_SkillsTab(t *testing.T) {
	t.Parallel()

	state := &StatusState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// プレイヤーを生成してスキルタブにデータがあることを確認
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	props := state.fetchProps(world)

	// スキルタブ
	skillTab := props.Tabs[2]
	assert.Equal(t, "skills", skillTab.ID)
	assert.Equal(t, len(gc.AllSkillIDs), len(skillTab.Items), "全スキルが表示される")
	assert.Equal(t, "刀剣", skillTab.Items[0].Label)
	assert.Equal(t, "0.000", skillTab.Items[0].Value)

	// 効果タブ
	effectTab := props.Tabs[3]
	assert.Equal(t, "effects", effectTab.ID)
	assert.NotEmpty(t, effectTab.Items, "効果項目がある")
}

func TestNewStatusState(t *testing.T) {
	t.Parallel()

	factory := NewStatusState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*StatusState)
	assert.True(t, ok, "StatusState型である")
}
