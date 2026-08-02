package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCharacterState_OnStartで各マウントを初期化する(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)

	require.NoError(t, state.OnStart(world))
	assert.NotNil(t, state.mount)
	assert.NotNil(t, state.windowMount)
	assert.NotNil(t, state.equipMount)
	assert.Equal(t, charSubBrowse, state.subState, "初期状態は閲覧")
}

func TestCharacterState_装備スロットは武器5と防具7の合計12(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	props := state.fetchProps(world)
	assert.Len(t, props.EquipSlots, 12, "武器5スロットと防具7スロット")
}

func TestCharacterState_スキル一覧はカテゴリ見出しを含む(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	props := state.fetchProps(world)
	hasHeader := false
	for _, row := range props.Skills {
		if row.IsHeader {
			hasHeader = true
			break
		}
	}
	assert.True(t, hasHeader, "スキル一覧はカテゴリ見出し行を持つ")
}
