package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
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

	t.Run("リーダーの隊員一覧を取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		m1, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)
		m2, err := lifecycle.SpawnSquadMember(world, leader, "隊員B", testAbilities(), "player")
		require.NoError(t, err)

		members := query.SquadMembers(world, leader)
		assert.Len(t, members, 2)
		assert.Contains(t, members, m1)
		assert.Contains(t, members, m2)
	})

	t.Run("隊員がいない場合は空を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		members := query.SquadMembers(world, leader)
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

		members := query.SquadMembers(world, leader)
		assert.Len(t, members, 1)
		assert.Contains(t, members, alive)
		assert.NotContains(t, members, dead)
	})

	t.Run("別のリーダーの隊員は含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader1, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// leader2はリーダー用のエンティティとして手動作成
		leader2 := world.Manager.NewEntity()
		leader2.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		_, err = lifecycle.SpawnSquadMember(world, leader1, "隊員1A", testAbilities(), "player")
		require.NoError(t, err)
		_, err = lifecycle.SpawnSquadMember(world, leader2, "隊員2A", testAbilities(), "player")
		require.NoError(t, err)

		members := query.SquadMembers(world, leader1)
		assert.Len(t, members, 1)
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

		memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)
		found, ok := query.SquadMemberAt(world, leader, int(memberGrid.X), int(memberGrid.Y))
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

		_, ok := query.SquadMemberAt(world, leader, 99, 99)
		assert.False(t, ok)
	})

	t.Run("別リーダーの隊員は見つからない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader1, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		leader2 := world.Manager.NewEntity()
		leader2.AddComponent(world.Components.GridElement, &gc.GridElement{X: 15, Y: 15})

		member2, err := lifecycle.SpawnSquadMember(world, leader2, "他隊員", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member2).(*gc.GridElement)
		_, ok := query.SquadMemberAt(world, leader1, int(memberGrid.X), int(memberGrid.Y))
		assert.False(t, ok, "別リーダーの隊員は見つからない")
	})

}

func TestSquadMemberCount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	assert.Equal(t, 0, query.SquadMemberCount(world, leader))

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)
	assert.Equal(t, 1, query.SquadMemberCount(world, leader))

	_, err = lifecycle.SpawnSquadMember(world, leader, "隊員B", testAbilities(), "player")
	require.NoError(t, err)
	assert.Equal(t, 2, query.SquadMemberCount(world, leader))
}

func TestSquadPolicy(t *testing.T) {
	t.Parallel()

	t.Run("隊員のポリシーを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		policy := query.SquadPolicy(world, member)
		assert.Equal(t, gc.PolicyEscort, policy.Position)
		assert.Equal(t, gc.PolicyAttack, policy.Combat)
	})

	t.Run("隊員でない場合はデフォルト値を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		nonMember := world.Manager.NewEntity()
		policy := query.SquadPolicy(world, nonMember)
		assert.Equal(t, gc.DefaultSquadPolicy(), policy)
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

func TestSquadLeader(t *testing.T) {
	t.Parallel()

	t.Run("隊員のリーダーを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		assert.Equal(t, leader, query.SquadLeader(world, member))
	})

	t.Run("隊員でない場合はゼロ値を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		nonMember := world.Manager.NewEntity()
		assert.Equal(t, ecs.Entity(0), query.SquadLeader(world, nonMember))
	})
}
