package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

// advanceHunger は turns ターン分だけ空腹を進め、各ターンで TurnNumber を1つ進める。
// progressTurnHunger は turn を撹拌のシードに使うので、ターンを止めたまま呼ぶと同じ結果が続いてしまう。
func advanceHunger(world w.World, turns int) {
	state := query.GetTurnState(world)
	for range turns {
		progressTurnHunger(world)
		state.TurnNumber++
	}
}

func TestProgressTurnHunger_空腹進行が基準ターン数に緩和される(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーではない Hunger 保持エンティティ。行動種別に依らずターン終了で全員進む
	actor := world.ECS.NewEntity()
	// 0 で止まるクランプを避けるため十分大きなプールにする
	world.Components.Hunger.Add(actor, &gc.Hunger{Current: 1_000_000, Max: 1_000_000})

	const turns = 3000
	advanceHunger(world, turns)

	// Add は値をコピーするためストレージから読み直す
	drained := 1_000_000 - world.Components.Hunger.Get(actor).Current
	expected := turns / gc.HungerDrainTurns // 100%進行での期待減少
	assert.InDelta(t, expected, drained, float64(expected)*0.15,
		"空腹進行は約 1/%d ターンに緩和されている", gc.HungerDrainTurns)
	assert.Less(t, drained, turns, "毎ターン減より明確に緩やか")
}

func TestProgressTurnHunger_同じ盤面なら決定的に進む(t *testing.T) {
	t.Parallel()

	// 共有 RNG を使わず entity と turn の hash でゲートするので、同じ盤面なら空腹の進みも一意に決まる。
	run := func() int {
		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()
		world.Components.Hunger.Add(actor, &gc.Hunger{Current: 1000, Max: 1000})
		advanceHunger(world, 100)
		return world.Components.Hunger.Get(actor).Current
	}

	first := run()
	second := run()
	assert.Equal(t, first, second, "同じ seed・盤面なら空腹の進みは一意")
}
