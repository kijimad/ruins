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

func TestGetSpatialIndex_隊員はCharactersに含まれない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	// SpatialIndexを再構築させる
	query.InvalidateSpatialIndex(world)
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)

	memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)
	leaderGrid := world.Components.GridElement.Get(leader).(*gc.GridElement)

	// リーダーはCharactersに含まれる
	_, leaderInChars := si.Characters[*leaderGrid]
	assert.True(t, leaderInChars, "リーダーはCharactersに含まれる")

	// 隊員はCharactersに含まれない
	_, memberInChars := si.Characters[*memberGrid]
	assert.False(t, memberInChars, "隊員はCharactersに含まれない")

	// 隊員はSquadMembersに含まれる
	_, memberInSquad := si.SquadMembers[*memberGrid]
	assert.True(t, memberInSquad, "隊員はSquadMembersに含まれる")
}

func TestInvalidateSpatialIndex_SquadMembersもクリアされる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	// 一度構築させる
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	assert.True(t, si.Built)
	assert.NotNil(t, si.SquadMembers)

	// 無効化する
	query.InvalidateSpatialIndex(world)

	// 無効化後はBuiltがfalse、各マップがnilになる
	si2 := query.GetSingleton[gc.SpatialIndex](world, world.Components.SpatialIndex)
	assert.False(t, si2.Built)
	assert.Nil(t, si2.SquadMembers)
	assert.Nil(t, si2.Characters)
	assert.Nil(t, si2.BlockPass)
	assert.Nil(t, si2.PlayerEntity)
}

func TestGetSpatialIndex_隊員のBlockPassはインデックスに登録される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	query.InvalidateSpatialIndex(world)
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)

	memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)

	// 隊員はBlockPassコンポーネントを持つので、BlockPassマップに含まれる
	assert.True(t, si.IsBlockPass(int(memberGrid.X), int(memberGrid.Y)),
		"隊員のBlockPassはインデックスに登録される")
}
