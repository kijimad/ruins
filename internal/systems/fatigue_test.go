package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProgressTurnFatigue_起床中は蓄積する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 0, Max: 2000})

	const turns = 100
	for range turns {
		progressTurnFatigue(world)
	}

	assert.Equal(t, turns*gc.FatigueGainPerTurn, world.Components.Fatigue.Get(actor).Current,
		"起床中は毎ターン蓄積する")
}

func TestProgressTurnFatigue_起床の蓄積はMaxで頭打ちになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1995, Max: 2000})

	for range 100 {
		progressTurnFatigue(world)
	}

	assert.Equal(t, 2000, world.Components.Fatigue.Get(actor).Current, "Max を超えない")
}

func TestProgressTurnFatigue_睡眠中は寝具品質に比例して減る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1000, Max: 2000})
	world.Components.Sleeping.Add(actor, &gc.Sleeping{Quality: consts.PercentBase})

	progressTurnFatigue(world)

	assert.Equal(t, 1000-fatigueRecoverPerTurn, world.Components.Fatigue.Get(actor).Current,
		"地べた品質100では基準量ぶん減る")
}

func TestProgressTurnFatigue_良い寝具は速く減る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1000, Max: 2000})
	world.Components.Sleeping.Add(actor, &gc.Sleeping{Quality: 150})

	progressTurnFatigue(world)

	assert.Equal(t, 1000-fatigueRecoverPerTurn*3/2, world.Components.Fatigue.Get(actor).Current,
		"品質150では1.5倍速く減る")
}

func TestProgressTurnFatigue_睡眠中は0でクランプする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 3, Max: 2000})
	world.Components.Sleeping.Add(actor, &gc.Sleeping{Quality: consts.PercentBase})

	progressTurnFatigue(world)

	assert.Equal(t, 0, world.Components.Fatigue.Get(actor).Current, "0 を下回らない")
}
