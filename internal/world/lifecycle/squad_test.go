package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
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

func TestSpawnSquadMember(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// リーダーを生成
	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// 隊員を生成
	member, err := SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	// 基本コンポーネントの確認
	assert.True(t, member.HasComponent(world.Components.Name), "Nameを持つ")
	assert.True(t, member.HasComponent(world.Components.Abilities), "Abilitiesを持つ")
	assert.True(t, member.HasComponent(world.Components.HP), "HPを持つ")
	assert.True(t, member.HasComponent(world.Components.Skills), "Skillsを持つ")
	assert.True(t, member.HasComponent(world.Components.CharModifiers), "CharModifiersを持つ")
	assert.True(t, member.HasComponent(world.Components.TurnBased), "TurnBasedを持つ")
	assert.True(t, member.HasComponent(world.Components.HealthStatus), "HealthStatusを持つ")
	assert.True(t, member.HasComponent(world.Components.GridElement), "GridElementを持つ")
	assert.True(t, member.HasComponent(world.Components.BlockPass), "BlockPassを持つ")

	// 隊員固有コンポーネントの確認
	assert.True(t, member.HasComponent(world.Components.SquadMember), "SquadMemberを持つ")
	assert.True(t, member.HasComponent(world.Components.SquadPolicy), "SquadPolicyを持つ")
	assert.True(t, member.HasComponent(world.Components.MemberAppearance), "MemberAppearanceを持つ")

	// AI系コンポーネントの確認
	assert.True(t, member.HasComponent(world.Components.AIMoveFSM), "AIMoveFSMを持つ")
	assert.True(t, member.HasComponent(world.Components.AIVision), "AIVisionを持つ")
	assert.True(t, member.HasComponent(world.Components.Disposition), "Dispositionを持つ")

	// ファクションの確認
	assert.True(t, member.HasComponent(world.Components.FactionAlly), "FactionAllyを持つ")
	assert.False(t, member.HasComponent(world.Components.FactionEnemy), "FactionEnemyは持たない")

	// プレイヤーマーカーは持たない
	assert.False(t, member.HasComponent(world.Components.Player), "Playerは持たない")

	// リーダー参照の確認
	sm := world.Components.SquadMember.Get(member).(*gc.SquadMember)
	assert.Equal(t, leader, sm.Leader, "リーダーへの参照が正しい")

	// デフォルトポリシーの確認
	policy := world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	assert.Equal(t, gc.PolicyEscort, policy.Position, "デフォルト位置ポリシーは護衛")
	assert.Equal(t, gc.PolicyAttack, policy.Combat, "デフォルト戦闘ポリシーは攻撃")
	assert.Equal(t, gc.PolicyPickup, policy.ItemPickup, "デフォルトアイテムポリシーは回収")
	assert.Equal(t, gc.PolicyDistribute, policy.ItemHandling, "デフォルトアイテム処理ポリシーは分配")

	// Dispositionの確認
	disp := world.Components.Disposition.Get(member).(*gc.Disposition)
	assert.Equal(t, gc.DispositionAlly, disp.Default, "デフォルト態度はAlly")
	assert.Equal(t, gc.DispositionAlly, disp.Current, "現在の態度はAlly")

	// 名前の確認
	name := world.Components.Name.Get(member).(*gc.Name)
	assert.Equal(t, "隊員A", name.Name)

	// HPが全回復していることの確認
	hp := world.Components.HP.Get(member).(*gc.HP)
	assert.Greater(t, hp.Max, 0, "最大HPが設定されている")
	assert.Equal(t, hp.Max, hp.Current, "HPが全回復している")
}

func TestSpawnSquadMember_リーダーと異なる位置に配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	leaderGrid := world.Components.GridElement.Get(leader).(*gc.GridElement)
	memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)

	// 隊員はリーダーと同じ位置に配置されない
	assert.False(t,
		leaderGrid.X == memberGrid.X && leaderGrid.Y == memberGrid.Y,
		"隊員はリーダーと異なる位置に配置される")
}

func TestSpawnSquadMember_リーダーにGridElementがないとエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// GridElementなしのエンティティ
	fakeLeader := world.Manager.NewEntity()

	_, err := SpawnSquadMember(world, fakeLeader, "隊員B", testAbilities(), "player")
	assert.Error(t, err, "GridElementなしのリーダーでスポーンするとエラー")
}

func TestDismissSquadMember(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員C", testAbilities(), "player")
	require.NoError(t, err)

	// 解雇前は存在する
	assert.True(t, member.HasComponent(world.Components.SquadMember))

	err = DismissSquadMember(world, member)
	require.NoError(t, err)
}

func TestDismissSquadMember_隊員でないエンティティはエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	nonMember := world.Manager.NewEntity()
	err := DismissSquadMember(world, nonMember)
	assert.Error(t, err)
}

func TestSetSquadPolicy(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員D", testAbilities(), "player")
	require.NoError(t, err)

	// ポリシー全体を変更
	newPolicy := gc.SquadPolicy{
		Position:     gc.PolicyVanguard,
		Combat:       gc.PolicyEvade,
		ItemPickup:   gc.PolicyIgnore,
		ItemHandling: gc.PolicyDistribute,
	}
	err = SetSquadPolicy(world, member, newPolicy)
	require.NoError(t, err)

	current := world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	assert.Equal(t, gc.PolicyVanguard, current.Position)
	assert.Equal(t, gc.PolicyEvade, current.Combat)
	assert.Equal(t, gc.PolicyIgnore, current.ItemPickup)
	assert.Equal(t, gc.PolicyDistribute, current.ItemHandling)
}

func TestSetSquadPolicy_位置だけ変更しても他のポリシーは変わらない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員E", testAbilities(), "player")
	require.NoError(t, err)

	policy := *world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	policy.Position = gc.PolicyPatrol
	err = SetSquadPolicy(world, member, policy)
	require.NoError(t, err)

	current := world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	assert.Equal(t, gc.PolicyPatrol, current.Position, "位置ポリシーが変更された")
	assert.Equal(t, gc.PolicyAttack, current.Combat, "戦闘ポリシーは変わらない")
}

func TestSpawnDefaultSquadMember(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnDefaultSquadMember(world, leader)
	require.NoError(t, err)

	// 基本コンポーネントが設定されている
	assert.True(t, member.HasComponent(world.Components.SquadMember), "SquadMemberを持つ")
	assert.True(t, member.HasComponent(world.Components.SquadPolicy), "SquadPolicyを持つ")
	assert.True(t, member.HasComponent(world.Components.Name), "Nameを持つ")

	// 名前が設定されている
	name := world.Components.Name.Get(member).(*gc.Name)
	assert.Equal(t, "Jim", name.Name)

	// リーダー参照が正しい
	sm := world.Components.SquadMember.Get(member).(*gc.SquadMember)
	assert.Equal(t, leader, sm.Leader)
	assert.True(t, sm.Active, "デフォルトで同行状態")
}

func TestSpawnDefaultSquadMember_リーダーにGridElementがないとエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	fakeLeader := world.Manager.NewEntity()
	_, err := SpawnDefaultSquadMember(world, fakeLeader)
	assert.Error(t, err)
}
