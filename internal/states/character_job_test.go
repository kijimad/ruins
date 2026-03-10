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

	assert.Equal(t, 4, len(professions), "職業は4種類")

	expectedIDs := []string{"evacuee", "soldier", "mechanic", "hunter"}
	for i, expectedID := range expectedIDs {
		assert.Equal(t, expectedID, professions[i].ID, "職業ID[%d]", i)
	}

	expectedNames := []string{"避難民", "軍人", "整備士", "猟師"}
	for i, expectedName := range expectedNames {
		assert.Equal(t, expectedName, professions[i].Name, "職業名[%d]", i)
	}
}

func TestProfessionItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		professionID string
		itemCount    int
	}{
		{professionID: "evacuee", itemCount: 2},
		{professionID: "soldier", itemCount: 3},
		{professionID: "mechanic", itemCount: 2},
		{professionID: "hunter", itemCount: 3},
	}

	for _, tt := range tests {
		t.Run(tt.professionID, func(t *testing.T) {
			t.Parallel()
			var found *Profession
			for i := range professions {
				if professions[i].ID == tt.professionID {
					found = &professions[i]
					break
				}
			}
			require.NotNil(t, found, "職業が見つからない: %s", tt.professionID)
			assert.Equal(t, tt.itemCount, len(found.Items), "初期アイテム数")
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

	props := state.fetchProps()

	assert.Equal(t, 4, len(props.Items), "職業は4つ")
	assert.Equal(t, "避難民", props.Items[0].Profession.Name)
	assert.Equal(t, "軍人", props.Items[1].Profession.Name)
	assert.Equal(t, "整備士", props.Items[2].Profession.Name)
	assert.Equal(t, "猟師", props.Items[3].Profession.Name)
}

func TestCharacterJobState_Navigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "job", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 初期状態
	itemIndex, ok := hooks.GetState[int](state.menuMount, "job_itemIndex")
	assert.True(t, ok)
	assert.Equal(t, 0, itemIndex, "初期インデックスは0")

	// 下に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()
	itemIndex, _ = hooks.GetState[int](state.menuMount, "job_itemIndex")
	assert.Equal(t, 1, itemIndex, "下移動後は1")

	// 上に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuUp)
	state.menuMount.Update()
	itemIndex, _ = hooks.GetState[int](state.menuMount, "job_itemIndex")
	assert.Equal(t, 0, itemIndex, "上移動後は0")
}

func TestCharacterJobState_CircularNavigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "job", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 最初から上に移動すると最後に
	state.menuMount.Dispatch(inputmapper.ActionMenuUp)
	state.menuMount.Update()
	itemIndex, _ := hooks.GetState[int](state.menuMount, "job_itemIndex")
	assert.Equal(t, 3, itemIndex, "循環して最後の項目に移動")

	// 最後から下に移動すると最初に
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()
	itemIndex, _ = hooks.GetState[int](state.menuMount, "job_itemIndex")
	assert.Equal(t, 0, itemIndex, "循環して最初の項目に移動")
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
