package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	iw "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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
}
