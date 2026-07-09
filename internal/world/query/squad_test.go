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

func testAbilities() gc.Abilities {
	return gc.Abilities{
		Vitality:  gc.Ability{Base: 10},
		Strength:  gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7},
		Dexterity: gc.Ability{Base: 6},
		Agility:   gc.Ability{Base: 9},
		Defense:   gc.Ability{Base: 5},
	}
}

func TestSquadMembers(t *testing.T) {
	t.Parallel()

	t.Run("隊員一覧を取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		m1, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)
		m2, err := lifecycle.SpawnSquadMember(world, leader, "隊員B", testAbilities(), "player")
		require.NoError(t, err)

		members := query.SquadMembers(world)
		assert.Len(t, members, 2)
		assert.Contains(t, members, m1)
		assert.Contains(t, members, m2)
	})

	t.Run("隊員がいない場合は空を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		members := query.SquadMembers(world)
		assert.Empty(t, members)
	})

	t.Run("死亡した隊員は除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		alive, err := lifecycle.SpawnSquadMember(world, leader, "生存者", testAbilities(), "player")
		require.NoError(t, err)
		dead, err := lifecycle.SpawnSquadMember(world, leader, "死亡者", testAbilities(), "player")
		require.NoError(t, err)

		dead.AddComponent(world.Components.Dead, &gc.Dead{})

		members := query.SquadMembers(world)
		assert.Len(t, members, 1)
		assert.Contains(t, members, alive)
		assert.NotContains(t, members, dead)
	})
}

func TestSquadMemberAt(t *testing.T) {
	t.Parallel()

	t.Run("指定座標にいる隊員を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.MustGet(member)
		found, ok := query.SquadMemberAt(world, int(memberGrid.X), int(memberGrid.Y))
		assert.True(t, ok)
		assert.Equal(t, member, found)
	})

	t.Run("指定座標に隊員がいない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		_, ok := query.SquadMemberAt(world, 99, 99)
		assert.False(t, ok)
	})
}

func TestSquadMemberCount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	assert.Equal(t, 0, query.SquadMemberCount(world))

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)
	assert.Equal(t, 1, query.SquadMemberCount(world))

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員B", testAbilities(), "player")
	require.NoError(t, err)
	assert.Equal(t, 2, query.SquadMemberCount(world))
}

func TestGetAI(t *testing.T) {
	t.Parallel()

	t.Run("隊員のAIを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		ai := query.GetAI(world, member)
		require.NotNil(t, ai)
		squadAI, ok := ai.Planner.(*gc.SquadAI)
		require.True(t, ok)
		assert.Equal(t, gc.SquadEscort, squadAI.Movement)
		assert.Equal(t, gc.CombatAttack, squadAI.CombatCurrent)
	})

	t.Run("AIがないエンティティではnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		nonMember := world.Manager.NewEntity()
		ai := query.GetAI(world, nonMember)
		assert.Nil(t, ai)
	})
}

func TestIsSquadMember(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	assert.True(t, query.IsSquadMember(world, member))
	assert.False(t, query.IsSquadMember(world, leader))

	nonMember := world.Manager.NewEntity()
	assert.False(t, query.IsSquadMember(world, nonMember))
}
