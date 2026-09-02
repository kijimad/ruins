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

	t.Run("満腹はボーナス", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 100, Max: 100})

		// 100 + 満腹20 = 120
		assert.Equal(t, consts.Percent(120), Metabolism(world, entity))
	})

	t.Run("標準の満腹度は増減なし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 80, Max: 100})

		assert.Equal(t, consts.Percent(100), Metabolism(world, entity))
	})

	t.Run("空腹はペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 50, Max: 100})

		// 100 - 空腹30 = 70
		assert.Equal(t, consts.Percent(70), Metabolism(world, entity))
	})

	t.Run("飢餓は大きなペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 20, Max: 100})

		// 100 - 飢餓60 = 40
		assert.Equal(t, consts.Percent(40), Metabolism(world, entity))
	})

	t.Run("VITと満腹度は合算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{Vitality: gc.Ability{Total: 10}})
		world.Components.Hunger.Add(entity, &gc.Hunger{Current: 20, Max: 100})

		// 100 + VIT*3(30) - 飢餓60 = 70
		assert.Equal(t, consts.Percent(70), Metabolism(world, entity))
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
}
