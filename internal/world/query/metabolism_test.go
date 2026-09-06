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

	t.Run("空腹は意識低下ぶん回復が下がる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 50, Max: 100})

		// 空腹は意識を10下げる。100 - 10 = 90
		assert.Equal(t, consts.Percent(90), Metabolism(world, entity))
	})

	t.Run("飢餓は意識をさらに下げる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 20, Max: 100})

		// 飢餓は意識を20下げる。100 - 20 = 80
		assert.Equal(t, consts.Percent(80), Metabolism(world, entity))
	})

	t.Run("VITと意識低下は合算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 10}})
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 20, Max: 100})

		// 100 + VIT*3(30) - 飢餓の意識低下20 = 110
		assert.Equal(t, consts.Percent(110), Metabolism(world, entity))
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

	t.Run("疲労で回復が下がる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		// 過労: 80%以上
		world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})

		// 過労は意識を25下げる。100 - 25 = 75
		assert.Equal(t, consts.Percent(75), Metabolism(world, entity))
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
