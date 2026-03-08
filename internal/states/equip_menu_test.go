package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.slotMount, "slotMountが初期化されている")
	assert.NotNil(t, state.windowMount, "windowMountが初期化されている")
	assert.NotNil(t, state.equipMount, "equipMountが初期化されている")
	assert.Equal(t, subStateSlotSelect, state.subState, "初期状態はスロット選択")
}

func TestEquipMenuState_FetchSlotProps(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchSlotProps(world)

	assert.Equal(t, 1, len(props.Tabs), "タブは1つ（装備）")
	assert.Equal(t, "player_equipment", props.Tabs[0].ID)
}

func TestEquipMenuState_TabNavigation(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchSlotProps(world)
	state.slotMount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	ui.UseTabMenu(state.slotMount.Store(), "slot", ui.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	state.slotMount.Update()

	// 初期状態
	tabIndex, _ := ui.GetState[int](state.slotMount, "slot_tabIndex")
	assert.Equal(t, 0, tabIndex, "初期タブインデックスは0")
}

func TestEquipMenuState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestEquipMenuState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "CloseMenuでTransPop")
}

func TestEquipMenuState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ナビゲーションアクションはTransNoneを返す
	actions := []inputmapper.ActionID{
		inputmapper.ActionMenuUp,
		inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft,
		inputmapper.ActionMenuRight,
	}

	for _, action := range actions {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
}

func TestEquipMenuState_SubState(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 初期状態ではスロット選択モード
	assert.Equal(t, subStateSlotSelect, state.subState, "初期状態ではスロット選択モード")
}

func TestEquipMenuState_DoAction_ActionWindow(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// アクションウィンドウを開く
	state.subState = subStateActionWindow
	state.windowMount.SetProps(windowScreenProps{
		SlotData: equipItemData{
			SlotLabel:  "武器1",
			SlotNumber: gc.SlotWeapon1,
		},
	})

	// ウィンドウモードでのキャンセル
	transition, err := state.DoAction(world, inputmapper.ActionWindowCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "ウィンドウキャンセルはTransNone")

	// スロット選択に戻る
	assert.Equal(t, subStateSlotSelect, state.subState, "キャンセル後はスロット選択モード")
}

func TestEquipMenuState_DoAction_ActionWindowNavigation(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// アクションウィンドウを開く
	state.subState = subStateActionWindow
	state.windowMount.SetProps(windowScreenProps{
		SlotData: equipItemData{
			SlotLabel:  "武器1",
			SlotNumber: gc.SlotWeapon1,
		},
	})

	// ウィンドウ用のUseStateを登録
	state.setupWindowState(world)
	state.windowMount.Update()

	// ウィンドウモードでの上下移動
	transition, err := state.DoAction(world, inputmapper.ActionWindowDown)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type)

	transition, err = state.DoAction(world, inputmapper.ActionWindowUp)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type)
}

func TestEquipMenuState_DoAction_EquipSelect(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 装備選択モードに切り替え
	state.subState = subStateEquipSelect
	state.equipMount.SetProps(equipScreenProps{
		SlotNumber: gc.SlotWeapon1,
	})

	// 装備選択モードでのキャンセル
	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "装備選択キャンセルはTransNone")

	// スロット選択に戻る
	assert.Equal(t, subStateSlotSelect, state.subState, "キャンセル後はスロット選択モード")
}

func TestEquipMenuState_DoAction_EquipSelectNavigation(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 装備選択モードに切り替え
	state.subState = subStateEquipSelect
	state.equipMount.SetProps(equipScreenProps{
		SlotNumber: gc.SlotWeapon1,
	})

	// ナビゲーションアクションはTransNoneを返す
	actions := []inputmapper.ActionID{
		inputmapper.ActionMenuUp,
		inputmapper.ActionMenuDown,
	}

	for _, action := range actions {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
}

func TestNewEquipMenuState(t *testing.T) {
	t.Parallel()

	factory := NewEquipMenuState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*EquipMenuState)
	assert.True(t, ok, "EquipMenuState型である")
}

func TestEquipMenuState_GetActionItems(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 空のアイテムの場合は閉じるのみ
	actions := state.getActionItems(world, equipItemData{})
	assert.Equal(t, []string{TextClose}, actions, "空のアイテムは閉じるのみ")

	// スロットラベルがあるアイテムの場合
	slotActions := state.getActionItems(world, equipItemData{
		SlotLabel:  "武器1",
		SlotNumber: gc.SlotWeapon1,
	})
	assert.Contains(t, slotActions, TextClose, "閉じるオプションがある")
}

func TestEquipMenuState_HandleInput(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// スロット選択モードではメニュー入力を使用
	state.subState = subStateSlotSelect
	_, _ = state.HandleInput(world.Config)

	// 装備選択モードでもメニュー入力を使用
	state.subState = subStateEquipSelect
	_, _ = state.HandleInput(world.Config)

	// アクションウィンドウモードではウィンドウ入力を使用
	state.subState = subStateActionWindow
	_, _ = state.HandleInput(world.Config)
}

func TestEquipMenuState_String(t *testing.T) {
	t.Parallel()

	state := &EquipMenuState{}
	assert.Equal(t, "EquipMenu", state.String())
}
