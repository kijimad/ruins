package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
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

func TestProfessionSelectionNavigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	state.initMenu(world)

	assert.Equal(t, 0, state.menuView.GetCurrentItemIndex(), "初期フォーカスは0")

	require.NoError(t, state.menuView.DoAction(inputmapper.ActionMenuDown))
	assert.Equal(t, 1, state.menuView.GetCurrentItemIndex(), "下移動後は1")

	require.NoError(t, state.menuView.DoAction(inputmapper.ActionMenuUp))
	assert.Equal(t, 0, state.menuView.GetCurrentItemIndex(), "上移動後は0")
}

func TestProfessionSelectionCircularNavigation(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	state.initMenu(world)

	currentTab := state.menuView.GetCurrentTab()
	itemCount := len(currentTab.Items)
	require.NoError(t, state.menuView.SetItemIndex(itemCount-1))

	require.NoError(t, state.menuView.DoAction(inputmapper.ActionMenuDown))
	assert.Equal(t, 0, state.menuView.GetCurrentItemIndex(), "循環移動で最初に戻る")
}

func TestProfessionSelectionCancel(t *testing.T) {
	t.Parallel()

	state := &CharacterJobState{playerName: "TestPlayer"}
	world := testutil.InitTestWorld(t)
	state.initMenu(world)

	require.NoError(t, state.menuView.DoAction(inputmapper.ActionMenuCancel))

	transition := state.GetTransition()
	require.NotNil(t, transition, "トランジションが設定されている")
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestNewCharacterJobState(t *testing.T) {
	t.Parallel()

	playerName := "TestPlayer"
	factory := NewCharacterJobState(playerName)
	state := factory().(*CharacterJobState)

	assert.Equal(t, playerName, state.playerName, "プレイヤー名が設定される")
}
