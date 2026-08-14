package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyHealing(t *testing.T) {
	t.Parallel()

	t.Run("HPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 回復対象を実スポーンの敵にする。HPとGridElementは生成時に備わる
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		hp := world.Components.HP.Get(entity)
		hp.Max = 100
		hp.Current = 50

		actual := ApplyHealing(world, entity, 30)
		assert.Equal(t, 30, actual)

		assert.Equal(t, 80, world.Components.HP.Get(entity).Current)
	})

	t.Run("最大HPを超えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		hp := world.Components.HP.Get(entity)
		hp.Max = 100
		hp.Current = 90

		actual := ApplyHealing(world, entity, 50)
		assert.Equal(t, 10, actual, "実際の回復量は10のみ")

		assert.Equal(t, 100, world.Components.HP.Get(entity).Current)
	})

	t.Run("HP満タンなら回復量は0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// SpawnEnemy が FullRecover で満タンにするため、満タン状態は手で作らない
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)

		actual := ApplyHealing(world, entity, 10)
		assert.Equal(t, 0, actual)
	})
}

func TestReactToHostileAction(t *testing.T) {
	t.Parallel()

	// AIの状態遷移のみを検証する純ロジックのため、対応するスポーンを介さず
	// SoloAI だけを手付与したエンティティで確認する。
	t.Run("CombatIgnoreはCombatAttackに変化する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.SoloAI.Add(entity, &gc.SoloAI{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore})

		reactToHostileAction(world, entity)

		solo := world.Components.SoloAI.Get(entity)
		assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
		assert.Equal(t, gc.CombatIgnore, solo.CombatDefault)
	})

	t.Run("CombatEvadeは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.SoloAI.Add(entity, &gc.SoloAI{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade})

		reactToHostileAction(world, entity)

		solo := world.Components.SoloAI.Get(entity)
		assert.Equal(t, gc.CombatEvade, solo.CombatCurrent)
		assert.Equal(t, gc.CombatEvade, solo.CombatDefault)
	})

	t.Run("CombatAttackは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.SoloAI.Add(entity, &gc.SoloAI{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack})

		reactToHostileAction(world, entity)

		solo := world.Components.SoloAI.Get(entity)
		assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
	})

	t.Run("AIがないエンティティではpanicしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()

		assert.NotPanics(t, func() {
			reactToHostileAction(world, entity)
		})
	})
}

func TestApplyDamage_Prop(t *testing.T) {
	t.Parallel()

	t.Run("ダメージでHPが減少する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)

		// crate は HP30 の破壊可能プロップ。実スポーンで Fixed と HP を備える
		prop, err := lifecycle.SpawnProp(world, "crate", 5, 5)
		require.NoError(t, err)

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 20, hp.Current)
		assert.False(t, world.Components.Dead.Has(prop))
	})

	t.Run("HPが0になるとDeadが付与される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)

		prop, err := lifecycle.SpawnProp(world, "crate", 5, 5)
		require.NoError(t, err)
		// 一撃で倒せるよう残HPを下げる
		world.Components.HP.Get(prop).Current = 10

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, world.Components.Dead.Has(prop))
	})

	t.Run("過剰ダメージでもHPは0で止まる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)

		prop, err := lifecycle.SpawnProp(world, "crate", 5, 5)
		require.NoError(t, err)
		world.Components.HP.Get(prop).Current = 5

		ApplyDamage(world, prop, 100, source)

		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, world.Components.Dead.Has(prop))
	})
}
