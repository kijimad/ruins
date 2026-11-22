package activity

import (
	"fmt"
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	iw "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThrowBehavior_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		item    string
		setup   func(world iw.World, enemy, item ecs.Entity)
		wantErr error
	}{
		{name: "射程内の敵に投げられる", item: "stone"},
		{name: "5 kg を超える物は投げられない", item: "deployable_bed", wantErr: errThrowTooHeavy},
		{
			name: "バックパックから消えた物は投げられない",
			item: "stone",
			setup: func(world iw.World, _, item ecs.Entity) {
				lifecycle.MoveMembersToField(world, []ecs.Entity{item}, consts.Coord[consts.Tile]{X: 1, Y: 1}, ecs.Entity{})
			},
			wantErr: errThrowItemNotInBackpack,
		},
		{
			name: "射線が遮られると投げられない",
			item: "stone",
			setup: func(world iw.World, _, _ ecs.Entity) {
				wall := world.ECS.NewEntity()
				world.Components.GridElement.Add(wall, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
				world.Components.BlockView.Add(wall, &gc.BlockView{})
			},
			wantErr: errThrowLineBlocked,
		},
		{
			name: "死んだ敵には投げられない",
			item: "stone",
			setup: func(world iw.World, enemy, _ ecs.Entity) {
				world.Components.Dead.Add(enemy, &gc.Dead{})
			},
			wantErr: errThrowTargetInvalid,
		},
		{
			name: "射程の外の敵には投げられない",
			item: "stone",
			setup: func(world iw.World, enemy, _ ecs.Entity) {
				world.Components.GridElement.Get(enemy).Coord = consts.Coord[consts.Tile]{X: 20, Y: 10}
			},
			wantErr: errThrowOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
			require.NoError(t, err)
			item, err := lifecycle.SpawnBackpackItem(world, tt.item, 1)
			require.NoError(t, err)
			if tt.setup != nil {
				tt.setup(world, enemy, item)
			}
			err = (&ThrowBehavior{}).Validate(NewThrowActivity(item, enemy), player, world)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestThrowBehavior_DoTurn_投げた物は無くなる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	stone, err := lifecycle.SpawnBackpackItem(world, "stone", 1)
	require.NoError(t, err)
	comp := NewThrowActivity(stone, enemy)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	assert.False(t, world.ECS.Alive(stone), "当たっても外れても無くなる")
	assert.True(t, world.Components.WeightDirty.Has(player), "持ち主の重さを計算し直す")
}

func TestThrowBehavior_DoTurn_構えた武器の残量は減らない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	require.NoError(t, lifecycle.UnequipAll(world, player))
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, sword, player, gc.SlotHand1)
	query.GetHandSelection(world).Slot = gc.SlotHand1
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	stone, err := lifecycle.SpawnBackpackItem(world, "stone", 1)
	require.NoError(t, err)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(stone, enemy), player, world))
	assert.Equal(t, 150, world.Components.Charge.Get(sword).Current)
}

func TestGrowWeaponSkill_投げると投擲のスキルが伸びる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)
	world.Components.Abilities.Add(actor, &gc.Abilities{})

	before := skills.Get(rawschema.SkillIdThrow).Exp.Current
	growWeaponSkill(actor, world, gc.ThrowAttack{Damage: 6, Accuracy: 70})
	assert.Greater(t, skills.Get(rawschema.SkillIdThrow).Exp.Current, before)
}

func TestThrowBehavior_DoTurn_積み重なった物は1つだけ減る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	grenade, err := lifecycle.SpawnBackpackItem(world, "grenade", 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "grenade", 1)
	require.NoError(t, err)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(grenade, enemy), player, world))
	assert.Equal(t, 1, countBackpackByRawID(world, player, "grenade"))
}

func TestThrowBehavior_DoTurn_威力0の物は傷を負わせずスキルも伸ばさない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	smoke, err := lifecycle.SpawnBackpackItem(world, "smoke_bomb", 1)
	require.NoError(t, err)
	hp := world.Components.HP.Get(enemy).Current
	exp := world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(smoke, enemy), player, world))

	assert.Equal(t, hp, world.Components.HP.Get(enemy).Current)
	assert.Equal(t, exp, world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current)
	assert.False(t, world.ECS.Alive(smoke), "投げた物は無くなる")
	assert.Equal(t, []string{"Ash threw Smoke Bomb."}, query.GetGameLog(world).GetRecent(1))
}

// alwaysLowSource は出目を常に最小にする乱数源。命中の判定を必ず当たりにする。
// 0 を返すと IntN が引き直しを続けて終わらないので 1 を返す
type alwaysLowSource struct{}

func (alwaysLowSource) Uint64() uint64 { return 1 }

func TestThrowBehavior_DoTurn_当たると傷を負わせ物は無くなる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.RNG = rand.New(alwaysLowSource{})
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	stone, err := lifecycle.SpawnBackpackItem(world, "stone", 1)
	require.NoError(t, err)
	hp := world.Components.HP.Get(enemy).Current

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(stone, enemy), player, world))

	assert.Less(t, world.Components.HP.Get(enemy).Current, hp)
	assert.False(t, world.ECS.Alive(stone))
}

func TestThrowBehavior_DoTurn_爆風は範囲の者すべてに効き範囲の外には効かない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	target, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	beside, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 14, Y: 11}, "fireball")
	require.NoError(t, err)
	outside, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 15, Y: 10}, "fireball")
	require.NoError(t, err)
	grenade, err := lifecycle.SpawnBackpackItem(world, "grenade", 1)
	require.NoError(t, err)
	hp := func(e ecs.Entity) int { return world.Components.HP.Get(e).Current }
	targetHP, besideHP, outsideHP, playerHP := hp(target), hp(beside), hp(outside), hp(player)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(grenade, target), player, world))

	assert.Less(t, hp(target), targetHP)
	assert.Less(t, hp(beside), besideHP)
	assert.Equal(t, outsideHP, hp(outside))
	assert.Equal(t, playerHP, hp(player))
	assert.False(t, world.ECS.Alive(grenade))
}

func TestThrowBehavior_DoTurn_爆風は投げた本人も巻き込む(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "fireball")
	require.NoError(t, err)
	grenade, err := lifecycle.SpawnBackpackItem(world, "grenade", 1)
	require.NoError(t, err)
	playerHP := world.Components.HP.Get(player).Current

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(grenade, enemy), player, world))

	assert.Less(t, world.Components.HP.Get(player).Current, playerHP)
	assert.Contains(t, query.GetGameLog(world).GetRecent(3), fmt.Sprintf("Ash was caught in Grenade and took %d damage.", playerHP-world.Components.HP.Get(player).Current))
}

func TestThrowBehavior_DoTurn_自分の爆風で倒れても物を消して終える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "fireball")
	require.NoError(t, err)
	grenade, err := lifecycle.SpawnBackpackItem(world, "grenade", 1)
	require.NoError(t, err)
	world.Components.HP.Get(player).Current = 1
	comp := NewThrowActivity(grenade, enemy)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(comp, player, world))

	assert.True(t, world.Components.Dead.Has(player))
	assert.False(t, world.ECS.Alive(grenade))
	assert.Equal(t, gc.ActivityStateCompleted, comp.State)
}

func TestThrowBehavior_DoTurn_毒の爆風は毒を付け重ねない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	world.Components.HP.Get(enemy).Current = 1000
	world.Components.HP.Get(enemy).Max = 1000
	first, err := lifecycle.SpawnBackpackItem(world, "poison_grenade", 1)
	require.NoError(t, err)
	venom := gc.ConditionType("venom")

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(first, enemy), player, world))
	cond := world.Components.HealthStatus.Get(enemy).Parts[gc.BodyPartWholeBody].GetCondition(venom)
	require.NotNil(t, cond)
	timer := cond.Timer
	assert.Equal(t, "Fireball contracted Venom.", query.GetGameLog(world).GetRecent(1)[0])

	second, err := lifecycle.SpawnBackpackItem(world, "poison_grenade", 1)
	require.NoError(t, err)
	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(second, enemy), player, world))
	assert.InDelta(t, timer, world.Components.HealthStatus.Get(enemy).Parts[gc.BodyPartWholeBody].GetCondition(venom).Timer, 0, "2発目で進行度は延びない")
}

func TestExplode(t *testing.T) {
	t.Parallel()

	venom := gc.ConditionType("venom")
	t.Run("威力0でも爆風の不調は付く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
		require.NoError(t, err)
		hp := world.Components.HP.Get(enemy).Current

		attack := gc.ThrowAttack{Blast: &gc.Blast{Radius: 1, Condition: venom}}
		require.NoError(t, explode(world, player, enemy, attack, "test"))
		assert.Equal(t, hp, world.Components.HP.Get(enemy).Current)
		assert.NotNil(t, world.Components.HealthStatus.Get(enemy).Parts[gc.BodyPartWholeBody].GetCondition(venom))
	})
	t.Run("自分だけを傷つけてもスキルは伸びない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		exp := world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current

		attack := gc.ThrowAttack{Damage: 5, Blast: &gc.Blast{Radius: 1}}
		require.NoError(t, explode(world, player, player, attack, "test"))
		assert.Equal(t, exp, world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current)
	})
	t.Run("キューブだけを傷つけてもスキルは伸びない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 13, Y: 10})
		require.NoError(t, err)
		exp := world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current

		attack := gc.ThrowAttack{Damage: 5, Blast: &gc.Blast{Radius: 1}}
		require.NoError(t, explode(world, player, cube, attack, "test"))
		assert.Equal(t, exp, world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current)
	})
	t.Run("爆風で倒れた者には不調を付けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
		require.NoError(t, err)
		world.Components.HP.Get(enemy).Current = 1

		attack := gc.ThrowAttack{Damage: 10, Blast: &gc.Blast{Radius: 1, Condition: venom}}
		require.NoError(t, explode(world, player, enemy, attack, "test"))
		require.True(t, world.Components.Dead.Has(enemy))
		assert.Nil(t, world.Components.HealthStatus.Get(enemy).Parts[gc.BodyPartWholeBody].GetCondition(venom))
	})
	t.Run("ほかの者を傷つけるとスキルが伸びる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
		require.NoError(t, err)
		exp := world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current

		attack := gc.ThrowAttack{Damage: 5, Blast: &gc.Blast{Radius: 1}}
		require.NoError(t, explode(world, player, enemy, attack, "test"))
		assert.Greater(t, world.Components.Skills.Get(player).Get(rawschema.SkillIdThrow).Exp.Current, exp)
	})
}

func TestCalculateThrowHitRate_爆ぜる物は100(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)

	assert.Equal(t, 100, CalculateThrowHitRate(player, enemy, gc.ThrowAttack{Damage: 20, Accuracy: 10, Blast: &gc.Blast{Radius: 1}}, world))
}

func TestExplode_構えた道具は置物に効かせない(t *testing.T) {
	t.Parallel()

	t.Run("構えた合う道具があっても道具が必須の木は削れない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		require.NoError(t, lifecycle.UnequipAll(world, player))
		axe, err := lifecycle.SpawnBackpackItem(world, "axe", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, axe, player, gc.SlotHand1)
		query.GetHandSelection(world).Slot = gc.SlotHand1
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
		require.NoError(t, err)
		tree, err := lifecycle.SpawnProp(world, "tree_a", 13, 9)
		require.NoError(t, err)
		treeHP := world.Components.HP.Get(tree).Current

		require.NoError(t, explode(world, player, enemy, gc.ThrowAttack{Damage: 20, Accuracy: 75, Blast: &gc.Blast{Radius: 1}}, "Grenade"))

		assert.Equal(t, treeHP, world.Components.HP.Get(tree).Current)
	})

	t.Run("合う道具を構えていても道具のスキルは伸びない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		require.NoError(t, lifecycle.UnequipAll(world, player))
		wrench, err := lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, wrench, player, gc.SlotHand1)
		query.GetHandSelection(world).Slot = gc.SlotHand1
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
		require.NoError(t, err)
		crate, err := lifecycle.SpawnProp(world, "wooden_crate", 13, 9)
		require.NoError(t, err)
		world.Components.HP.Get(crate).Current = 1
		def, ok := raw.FindDisassembly(world.Resources.RawMaster, "wooden_crate")
		require.True(t, ok)
		skillID, ok := world.Resources.SkillDefs.ToolSkills[*def.ToolCategory]
		require.True(t, ok)
		before := *world.Components.Skills.Get(player).Get(skillID)

		require.NoError(t, explode(world, player, enemy, gc.ThrowAttack{Damage: 20, Accuracy: 75, Blast: &gc.Blast{Radius: 1}}, "Grenade"))

		require.True(t, world.Components.Dead.Has(crate))
		assert.Equal(t, before, *world.Components.Skills.Get(player).Get(skillID))
	})
}

func TestExplode_閃光弾は範囲の者の目をくらませる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 14, Y: 10}, "fireball")
	require.NoError(t, err)
	flashbang, err := lifecycle.SpawnBackpackItem(world, "flashbang", 1)
	require.NoError(t, err)
	attack, ok := query.ThrowAttackOf(world, flashbang)
	require.True(t, ok)
	hp := world.Components.HP.Get(enemy).Current
	speed := query.CalculateSpeed(world, enemy)
	before := query.EffectiveBodyFuncs(world, enemy)

	require.NoError(t, explode(world, player, enemy, attack, "Flashbang"))

	funcs := query.EffectiveBodyFuncs(world, enemy)
	assert.Equal(t, hp, world.Components.HP.Get(enemy).Current, "閃光弾は傷つけない")
	assert.Equal(t, before.Consciousness, funcs.Consciousness)
	assert.Equal(t, before.Moving, funcs.Moving)
	assert.Equal(t, before.Sight*60/100, funcs.Sight, "発症したての目くらみは軽度で、頭の視覚を 40 下げる")
	assert.Equal(t, speed, query.CalculateSpeed(world, enemy), "頭の不調は速さを変えない")
}

func TestExplode_閃光弾は半径2までの者の目を本人も含めてくらませる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	target, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "fireball")
	require.NoError(t, err)
	far, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 15, Y: 10}, "fireball")
	require.NoError(t, err)
	flashbang, err := lifecycle.SpawnBackpackItem(world, "flashbang", 1)
	require.NoError(t, err)
	attack, ok := query.ThrowAttackOf(world, flashbang)
	require.True(t, ok)

	require.NoError(t, explode(world, player, target, attack, "Flashbang"))

	daze := gc.ConditionType("daze")
	dazed := func(e ecs.Entity) bool {
		return world.Components.HealthStatus.Get(e).Parts[gc.BodyPartHead].GetCondition(daze) != nil
	}
	assert.Equal(t, []bool{true, true, false}, []bool{dazed(player), dazed(target), dazed(far)})
}
