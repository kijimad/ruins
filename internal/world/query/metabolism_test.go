package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetabolism(t *testing.T) {
	t.Parallel()

	t.Run("能力も満腹度もなければ基準100", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()

		assert.Equal(t, consts.Percent(100), Metabolism(world, entity))
	})

	t.Run("VITが高いほど速い", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 10}})

		// 100 + VIT*3 = 130
		assert.Equal(t, consts.Percent(130), Metabolism(world, entity))
	})

	t.Run("満腹でも増減なし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 100, Max: 100})

		// 満腹は意識を下げないので基準100のまま
		assert.Equal(t, consts.Percent(100), Metabolism(world, entity))
	})

	t.Run("標準の満腹度は増減なし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 80, Max: 100})

		assert.Equal(t, consts.Percent(100), Metabolism(world, entity))
	})

	t.Run("空腹の栄養失調は意識経由で回復を下げる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		// 空腹は Malnutrition 不調として意識を下げ、回復も funnel 経由で落ちる。軽度 8*1=8
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionMalnutrition, gc.SeverityMinor)
		world.Components.HealthStatus.Add(entity, hs)

		// 意識=100-8=92
		assert.Equal(t, consts.Percent(92), Metabolism(world, entity))
	})

	t.Run("飢餓は意識をさらに下げる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionMalnutrition, gc.SeveritySevere)
		world.Components.HealthStatus.Add(entity, hs)

		// 飢餓 8*3=24。意識=100-24=76
		assert.Equal(t, consts.Percent(76), Metabolism(world, entity))
	})

	t.Run("VITと意識低下は合算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 10}})
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionMalnutrition, gc.SeveritySevere)
		world.Components.HealthStatus.Add(entity, hs)

		// 意識76 + VIT*3(30) = 106
		assert.Equal(t, consts.Percent(106), Metabolism(world, entity))
	})

	t.Run("下限は0でマイナスにならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		// 健康状態の低下で VIT 合計が負になっても代謝は 0 で止まる
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: -40}})

		// 100 + (-40*3) = -20 → 0 にクランプ
		assert.Equal(t, consts.Percent(0), Metabolism(world, entity))
	})

	t.Run("過労は意識経由で回復を下げる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionExhaustion, gc.SeveritySevere)
		world.Components.HealthStatus.Add(entity, hs)

		// 過労 10*3=30。意識=100-30=70
		assert.Equal(t, consts.Percent(70), Metabolism(world, entity))
	})

	t.Run("睡眠中は回復が上がり寝具品質に比例する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Sleeping.Add(entity, &gc.Sleeping{Quality: 150})

		// 100 + 睡眠50*1.5 = 175
		assert.Equal(t, consts.Percent(175), Metabolism(world, entity))
	})
}
