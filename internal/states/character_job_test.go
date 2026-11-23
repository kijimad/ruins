package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfessions(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)

	assert.Len(t, professions, 7, "職業は7種類")

	expectedIDs := []string{"evacuee", "hunter", "mechanic", "medic", "sniper", "soldier", "debug"}
	for i, expectedID := range expectedIDs {
		assert.Equal(t, expectedID, professions[i].Id, "職業ID[%d]", i)
	}

	expectedNames := []string{"Refugee", "Hunter", "Engineer", "Medic", "Sniper", "Soldier", "Debug"}
	for i, expectedName := range expectedNames {
		assert.Equal(t, expectedName, professions[i].Name, "職業名[%d]", i)
	}
}

func TestProfessionItems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)

	tests := []struct {
		professionID string
		itemCount    int
	}{
		{professionID: "evacuee", itemCount: 4},
		{professionID: "soldier", itemCount: 1},
		{professionID: "sniper", itemCount: 3},
		{professionID: "mechanic", itemCount: 2},
		{professionID: "hunter", itemCount: 3},
		{professionID: "medic", itemCount: 4},
		{professionID: "debug", itemCount: 23},
	}

	for _, tt := range tests {
		t.Run(tt.professionID, func(t *testing.T) {
			t.Parallel()
			var found bool
			for _, p := range professions {
				if p.Id == tt.professionID {
					assert.Len(t, p.Items, tt.itemCount, "初期アイテム数")
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
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)

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
		{professionID: "debug", equipCount: 7},
	}

	for _, tt := range tests {
		t.Run(tt.professionID, func(t *testing.T) {
			t.Parallel()
			var found bool
			for _, p := range professions {
				if p.Id == tt.professionID {
					assert.Len(t, p.Equips, tt.equipCount, "初期装備数")
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

	require.NoError(t, state.OnStart(world))
}

func TestCharacterJobState_FetchProps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		debug bool
		want  []string
	}{
		{name: "Debug でないとき", debug: false, want: []string{"evacuee", "hunter", "mechanic", "medic", "sniper", "soldier"}},
		{name: "Debug のとき", debug: true, want: []string{"evacuee", "hunter", "mechanic", "medic", "sniper", "soldier", "debug"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := &CharacterJobState{playerName: "TestPlayer"}
			world := testutil.InitTestWorld(t)
			world.Resources.Config.Debug = tt.debug
			require.NoError(t, state.OnStart(world))

			props, err := state.Fetch(world)
			require.NoError(t, err)

			ids := make([]string, len(props.Items))
			for i, item := range props.Items {
				ids[i] = item.Profession.Id
			}
			assert.Equal(t, tt.want, ids)
		})
	}
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

func TestNewCharacterJobState(t *testing.T) {
	t.Parallel()

	playerName := "TestPlayer"
	factory := NewCharacterJobState(playerName)
	s, err := factory()
	require.NoError(t, err)
	state, ok := s.(*CharacterJobState)
	require.True(t, ok, "型が *CharacterJobState であるべき")

	assert.Equal(t, playerName, state.playerName, "プレイヤー名が設定される")
}
