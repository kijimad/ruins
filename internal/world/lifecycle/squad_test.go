package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
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
	assert.True(t, world.Components.Name.Has(member), "Nameを持つ")
	assert.True(t, world.Components.Abilities.Has(member), "Abilitiesを持つ")
	assert.True(t, world.Components.HP.Has(member), "HPを持つ")
	assert.True(t, world.Components.Skills.Has(member), "Skillsを持つ")
	assert.True(t, world.Components.CharModifiers.Has(member), "CharModifiersを持つ")
	assert.True(t, world.Components.TurnBased.Has(member), "TurnBasedを持つ")
	assert.True(t, world.Components.HealthStatus.Has(member), "HealthStatusを持つ")
	assert.True(t, world.Components.GridElement.Has(member), "GridElementを持つ")
	assert.False(t, world.Components.BlockPass.Has(member), "キャラクターはBlockPassを持たない")

	// 隊員固有コンポーネントの確認
	assert.True(t, world.Components.SquadMember.Has(member), "SquadMemberを持つ")

	// AIコンポーネントの確認
	assert.True(t, world.Components.SquadAI.Has(member), "AIを持つ")

	// ファクションの確認
	assert.True(t, query.IsAlly(world, member), "味方派閥に属する")
	assert.False(t, query.IsEnemy(world, member), "敵派閥には属さない")

	// プレイヤーマーカーは持たない
	assert.False(t, world.Components.Player.Has(member), "Playerは持たない")

	// デフォルトAIの確認
	squad := world.Components.SquadAI.Get(member)
	assert.Equal(t, gc.PlannerSquad, squad.Type(), "PlannerはSquad")
	assert.Equal(t, gc.SquadEscort, squad.Movement, "デフォルト移動ポリシーは護衛")
	assert.Equal(t, gc.CombatAttack, squad.CombatCurrent, "デフォルト戦闘ポリシーは攻撃")
	assert.Equal(t, gc.PolicyPickup, squad.ItemPickup, "デフォルトアイテムポリシーは回収")
	assert.Equal(t, gc.PolicyDistribute, squad.ItemHandling, "デフォルトアイテム処理ポリシーは分配")

	// 名前の確認
	name := world.Components.Name.Get(member)
	assert.Equal(t, "隊員A", name.Name)

	// HPが全回復していることの確認
	hp := world.Components.HP.Get(member)
	assert.Positive(t, hp.Max, "最大HPが設定されている")
	assert.Equal(t, hp.Max, hp.Current, "HPが全回復している")
}

func TestSpawnSquadMember_リーダーと異なる位置に配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員", testAbilities(), "player")
	require.NoError(t, err)

	leaderGrid := world.Components.GridElement.Get(leader)
	memberGrid := world.Components.GridElement.Get(member)

	// 隊員はリーダーと同じ位置に配置されない
	assert.False(t,
		leaderGrid.X == memberGrid.X && leaderGrid.Y == memberGrid.Y,
		"隊員はリーダーと異なる位置に配置される")
}

func TestSpawnSquadMember_リーダーにGridElementがないとエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// GridElementなしのエンティティ
	fakeLeader := world.ECS.NewEntity()

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
	assert.True(t, world.Components.SquadMember.Has(member))

	err = DismissSquadMember(world, member)
	require.NoError(t, err)
}

func TestDismissSquadMember_隊員でないエンティティはエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	nonMember := world.ECS.NewEntity()
	err := DismissSquadMember(world, nonMember)
	assert.Error(t, err)
}

func TestGetAI(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員D", testAbilities(), "player")
	require.NoError(t, err)

	// AIコンポーネントを取得して変更する
	squad, err := GetAI(world, member)
	require.NoError(t, err)

	squad.Movement = gc.SquadVanguard
	squad.CombatDefault = gc.CombatEvade
	squad.CombatCurrent = gc.CombatEvade
	squad.ItemPickup = gc.PolicyIgnore
	squad.ItemHandling = gc.PolicyDistribute

	currentSquad := world.Components.SquadAI.Get(member)
	assert.Equal(t, gc.SquadVanguard, currentSquad.Movement)
	assert.Equal(t, gc.CombatEvade, currentSquad.CombatCurrent)
	assert.Equal(t, gc.PolicyIgnore, currentSquad.ItemPickup)
	assert.Equal(t, gc.PolicyDistribute, currentSquad.ItemHandling)
}

func TestGetAI_移動だけ変更しても他のポリシーは変わらない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, leader, "隊員E", testAbilities(), "player")
	require.NoError(t, err)

	squad, err := GetAI(world, member)
	require.NoError(t, err)
	squad.Movement = gc.SquadPatrol

	currentSquad := world.Components.SquadAI.Get(member)
	assert.Equal(t, gc.SquadPatrol, currentSquad.Movement, "移動ポリシーが変更された")
	assert.Equal(t, gc.CombatAttack, currentSquad.CombatCurrent, "戦闘ポリシーは変わらない")
}

func TestSpawnDefaultSquadMember(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	leader, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnDefaultSquadMember(world, leader)
	require.NoError(t, err)

	// 基本コンポーネントが設定されている
	assert.True(t, world.Components.SquadMember.Has(member), "SquadMemberを持つ")
	assert.True(t, world.Components.SquadAI.Has(member), "AIを持つ")
	assert.True(t, world.Components.Name.Has(member), "Nameを持つ")

	// 名前が設定されている
	name := world.Components.Name.Get(member)
	assert.Equal(t, "Jim", name.Name)

	// 隊員マーカーがある
	assert.True(t, world.Components.SquadMember.Has(member))
}

func TestSpawnDefaultSquadMember_リーダーにGridElementがないとエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	fakeLeader := world.ECS.NewEntity()
	_, err := SpawnDefaultSquadMember(world, fakeLeader)
	assert.Error(t, err)
}
