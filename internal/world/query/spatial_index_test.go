package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSpatialIndex_キャラクターはBlockPassに含まれない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	query.InvalidateSpatialIndex(world)
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)

	memberGrid := world.Components.GridElement.MustGet(member)
	leaderGrid := world.Components.GridElement.MustGet(leader)

	assert.False(t, si.BlockPass[*leaderGrid], "プレイヤーはBlockPassに含まれない")
	assert.False(t, si.BlockPass[*memberGrid], "隊員はBlockPassに含まれない")

	_, leaderInChars := si.Characters[*leaderGrid]
	assert.True(t, leaderInChars, "プレイヤーはCharactersに含まれる")

	_, memberInChars := si.Characters[*memberGrid]
	assert.True(t, memberInChars, "隊員はCharactersに含まれる")
}

func TestInvalidateSpatialIndex_全マップがクリアされる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	assert.True(t, si.Built)
	assert.NotNil(t, si.Characters)

	query.InvalidateSpatialIndex(world)

	si2 := query.GetSingleton[gc.SpatialIndex](world, world.Components.SpatialIndex)
	assert.False(t, si2.Built)
	assert.Nil(t, si2.Characters)
	assert.Nil(t, si2.BlockPass)
	assert.Nil(t, si2.PlayerEntity)
}
