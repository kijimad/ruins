package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HealthRegenSystem は ActiveFilter1[HP] で回すだけなので裸エンティティで動作する。
// あえて SpawnPlayer を使わないのは、能力値と満腹度を持たせず代謝を確定的に100%へ固定し、
// 回復量を厳密に検証するため。回復は healthRegenIntervalTurns ターンに一度なので、回復を期待する
// ケースでは TurnNumber を回復ターンへ合わせる。
func TestHealthRegenSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("回復ターンには基準代謝ぶん回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		// 代謝100%なので per-interval(1) ぶん回復する
		assert.Equal(t, 11, world.Components.HP.Get(entity).Current)
	})

	t.Run("回復ターン以外は回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// interval で割り切れないターンでは間引かれて回復しない
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns - 1
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 10, world.Components.HP.Get(entity).Current)
	})

	t.Run("満タンは回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 30, Max: 30})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 30, world.Components.HP.Get(entity).Current)
	})

	t.Run("最大値を超えて回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 29, Max: 30})
		// 代謝220%で回復2。29 + 2 = 31 だが最大30でクランプ
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 40}})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 30, world.Components.HP.Get(entity).Current)
	})

	t.Run("死亡エンティティは回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 0, Max: 30})
		world.Components.Dead.Add(entity, &gc.Dead{})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 0, world.Components.HP.Get(entity).Current)
	})

	t.Run("飢餓は代謝を下げ回復ターンでも回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})
		// 飢餓は代謝 capacity を20下げて80%にする。per-interval(1)*0.8=0.8 は切り捨てで0
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 20, Max: 100})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 10, world.Components.HP.Get(entity).Current)
	})

	t.Run("高VITでは代謝が上がり回復が速い", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 40}})

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		// 代謝220%なので per-interval(1)*2.2=2.2 は切り捨てで2
		assert.Equal(t, 12, world.Components.HP.Get(entity).Current)
	})

	t.Run("失血中は自然回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})
		hs := &gc.HealthStatus{}
		// 重い切り傷は血液量を危険域まで下げる。失血を回復で打ち消さない
		hs.Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: 80, Severity: gc.TimerToSeverity(80)})
		world.Components.HealthStatus.Add(entity, hs)

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 10, world.Components.HP.Get(entity).Current)
	})

	t.Run("重症の低体温でも自然回復しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetTurnState(world).TurnNumber = healthRegenIntervalTurns
		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Current: 10, Max: 30})
		hs := &gc.HealthStatus{}
		// 出血だけでなく、重症の低体温のように HP を削る不調があるあいだは回復を止める
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{Type: gc.ConditionHypothermia, Timer: 90, Severity: gc.SeveritySevere})
		world.Components.HealthStatus.Add(entity, hs)

		require.NoError(t, (&HealthRegenSystem{}).Update(world))

		assert.Equal(t, 10, world.Components.HP.Get(entity).Current)
	})
}
