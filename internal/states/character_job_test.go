package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfessions(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	professions := world.Resources.RawMaster.Raws.Professions

	assert.Equal(t, 6, len(professions), "職業は6種類")

	expectedIDs := []string{"evacuee", "hunter", "mechanic", "medic", "sniper", "soldier"}
	for i, expectedID := range expectedIDs {
		assert.Equal(t, expectedID, professions[i].Id, "職業ID[%d]", i)
	}

	expectedNames := []string{"避難民", "猟師", "整備士", "衛生兵", "狙撃手", "軍人"}
	for i, expectedName := range expectedNames {
		assert.Equal(t, expectedName, professions[i].Name, "職業名[%d]", i)
	}
}

func TestProfessionItems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	professions := world.Resources.RawMaster.Raws.Professions

	tests := []struct {
		professionID string
		itemCount    int
	}{
		{professionID: "evacuee", itemCount: 3},
		{professionID: "soldier", itemCount: 1},
		{professionID: "sniper", itemCount: 3},
		{professionID: "mechanic", itemCount: 2},
		{professionID: "hunter", itemCount: 3},
		{professionID: "medic", itemCount: 5},
	}

	for _, tt := range tests {
		t.Run(tt.professionID, func(t *testing.T) {
			t.Parallel()
			var found bool
			for _, p := range professions {
				if p.Id == tt.professionID {
					assert.Equal(t, tt.itemCount, len(p.Items), "初期アイテム数")
					found = true
					break
				}
			}
			require.True(t, found, "職業が見つからない: %s", tt.professionID)
		})
	}
}

func TestProfessionEquips(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	professions := world.Resources.RawMaster.Raws.Professions

	tests := []struct {
		professionID string
		equipCount   int
	}{
		{professionID: "evacuee", equipCount: 6},
		{professionID: "soldier", equipCount: 7},
		{professionID: "sniper", equipCount: 7},
		{professionID: "mechanic", equipCount: 7},
		{professionID: "hunter", equipCount: 7},
		{professionID: "medic", equipCount: 6},
	}

	for _, tt := range tests {
		t.Run(tt.professionID, func(t *testing.T) {
			t.Parallel()
			var found bool
			for _, p := range professions {
				if p.Id == tt.professionID {
					assert.Equal(t, tt.equipCount, len(p.Equips), "初期装備数")
					found = true
					break
				}
			}
			require.True(t, found, "職業が見つからない: %s", tt.professionID)
		})
	}
}

func TestCharacterJobState_OnStart(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.menuMount, "menuMountが初期化されている")
}

func TestCharacterJobState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	assert.Equal(t, 6, len(props.Items), "職業は6つ")
	assert.Equal(t, "避難民", props.Items[0].Profession.Name)
	assert.Equal(t, "猟師", props.Items[1].Profession.Name)
	assert.Equal(t, "整備士", props.Items[2].Profession.Name)
	assert.Equal(t, "衛生兵", props.Items[3].Profession.Name)
	assert.Equal(t, "狙撃手", props.Items[4].Profession.Name)
	assert.Equal(t, "軍人", props.Items[5].Profession.Name)
}

func TestCharacterJobState_Navigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "job", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 初期状態
	menuState, ok := hooks.GetState[hooks.TabMenuState](state.menuMount, "job")
	assert.True(t, ok)
	assert.Equal(t, 0, menuState.ItemIndex, "初期インデックスは0")

	// 下に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()
	menuState, _ = hooks.GetState[hooks.TabMenuState](state.menuMount, "job")
	assert.Equal(t, 1, menuState.ItemIndex, "下移動後は1")

	// 上に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuUp)
	state.menuMount.Update()
	menuState, _ = hooks.GetState[hooks.TabMenuState](state.menuMount, "job")
	assert.Equal(t, 0, menuState.ItemIndex, "上移動後は0")
}

func TestCharacterJobState_CircularNavigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "job", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 最初から上に移動すると最後に
	state.menuMount.Dispatch(inputmapper.ActionMenuUp)
	state.menuMount.Update()
	menuState, _ := hooks.GetState[hooks.TabMenuState](state.menuMount, "job")
	assert.Equal(t, 5, menuState.ItemIndex, "循環して最後の項目に移動")

	// 最後から下に移動すると最初に
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()
	menuState, _ = hooks.GetState[hooks.TabMenuState](state.menuMount, "job")
	assert.Equal(t, 0, menuState.ItemIndex, "循環して最初の項目に移動")
}

func TestCharacterJobState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestCharacterJobState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "CloseMenuでTransPop")
}

func TestCharacterJobState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

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

func TestNewCharacterJobState(t *testing.T) {
	t.Parallel()

	playerName := "TestPlayer"
	factory := NewCharacterJobState(playerName)
	state := factory().(*CharacterJobState)

	assert.Equal(t, playerName, state.playerName, "プレイヤー名が設定される")
}

func TestCharacterJobState_HandleInput(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	_, _ = state.HandleInput(world.Config)
}

func TestCharacterJobState_String(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{}
	assert.Equal(t, "CharacterJob", state.String())
}
