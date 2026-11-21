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

// setupThrowWorld はプレイヤーと射程内の敵を置き、背負い袋に item を1つ持たせる
func setupThrowWorld(t *testing.T, item string) (world iw.World, player, enemy, thrown ecs.Entity) {
	t.Helper()
	world = testutil.InitTestWorld(t)
	p, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	e, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "fireball")
	require.NoError(t, err)
	it, err := lifecycle.SpawnBackpackItem(world, item, 1)
	require.NoError(t, err)
	return world, p, e, it
}

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
			name: "背負い袋から消えた物は投げられない",
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
			world, player, enemy, item := setupThrowWorld(t, tt.item)
			if tt.setup != nil {
				tt.setup(world, enemy, item)
			}
			err := (&ThrowBehavior{}).Validate(NewThrowActivity(item, enemy), player, world)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLandThrown(t *testing.T) {
	t.Parallel()

	t.Run("消える物は1つ減る", func(t *testing.T) {
		t.Parallel()
		world, player, _, stone := setupThrowWorld(t, "stone")
		_, err := lifecycle.SpawnBackpackItem(world, "stone", 1)
		require.NoError(t, err)

		require.NoError(t, landThrown(world, player, stone, consts.Coord[consts.Tile]{X: 13, Y: 10}, true))
		assert.Equal(t, 1, countBackpackByRawID(world, player, "stone"))
		assert.True(t, world.Components.WeightDirty.Has(player), "持ち主の重さを計算し直す")
	})
	t.Run("消えない物は落ちて拾い直せる", func(t *testing.T) {
		t.Parallel()
		world, player, _, stone := setupThrowWorld(t, "stone")
		coord := consts.Coord[consts.Tile]{X: 13, Y: 10}

		require.NoError(t, landThrown(world, player, stone, coord, false))
		assert.False(t, world.Components.LocationInBackpack.Has(stone))
		require.True(t, world.Components.GridElement.Has(stone))
		assert.Equal(t, coord, world.Components.GridElement.Get(stone).Coord)
		assert.True(t, world.Components.WeightDirty.Has(player), "持ち主の重さを計算し直す")
	})
}

func TestThrowBehavior_DoTurn_投げた物は背負い袋から出る(t *testing.T) {
	t.Parallel()

	world, player, enemy, stone := setupThrowWorld(t, "stone")
	comp := NewThrowActivity(stone, enemy)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	assert.Equal(t, 0, countBackpackByRawID(world, player, "stone"), "当たれば消え、外れれば敵の足元に落ちる")
}

func TestThrowBehavior_DoTurn_壊れる物は外れても無くなる(t *testing.T) {
	t.Parallel()

	world, player, enemy, grenade := setupThrowWorld(t, "grenade")
	comp := NewThrowActivity(grenade, enemy)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(comp, player, world))

	assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	assert.False(t, world.ECS.Alive(grenade))
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

func TestThrowBehavior_DoTurn_壊れる物は1つだけ減る(t *testing.T) {
	t.Parallel()

	world, player, enemy, grenade := setupThrowWorld(t, "grenade")
	_, err := lifecycle.SpawnBackpackItem(world, "grenade", 1)
	require.NoError(t, err)

	require.NoError(t, (&ThrowBehavior{}).DoTurn(NewThrowActivity(grenade, enemy), player, world))
	assert.Equal(t, 1, countBackpackByRawID(world, player, "grenade"))
}
