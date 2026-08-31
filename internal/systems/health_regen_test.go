package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HealthRegenSystem は ActiveFilter1[HP] で回すだけなので裸エンティティで動作する。
// あえて SpawnPlayer を使わないのは、能力値と満腹度を持たせず代謝を確定的に100%へ固定し、
// 回復量を厳密に検証するため。
func TestHealthRegenSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("基準代謝で毎ターンbaseぶん回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		// 代謝100%なので base(2) ぶん回復する
		assert.Equal(t, 12, world.Components.HP.Get(entity).Current)
	})

	t.Run("満タンは回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 30, Max: 30})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 30, world.Components.HP.Get(entity).Current)
	})

	t.Run("最大値を超えて回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 29, Max: 30})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		// 29 + base(2) = 31 だが最大30でクランプ
		assert.Equal(t, 30, world.Components.HP.Get(entity).Current)
	})

	t.Run("死亡エンティティは回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 0, Max: 30})
		world.Components.Dead.Add(entity, &gc.Dead{})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 0, world.Components.HP.Get(entity).Current)
	})

	t.Run("飢餓では代謝が0になり回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 20, Max: 100}) // 飢餓

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		// 代謝40%なので base(2)*0.4=0.8 は切り捨てで0。回復しない
		assert.Equal(t, 10, world.Components.HP.Get(entity).Current)
	})

	t.Run("高VITでは代謝が上がり回復が速い", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 20}})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		// 代謝160%なので base(2)*1.6=3.2 は切り捨てで3
		assert.Equal(t, 13, world.Components.HP.Get(entity).Current)
	})
}
