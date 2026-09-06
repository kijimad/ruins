package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressTurnFatigue_過労はExhaustion不調を立てる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1600, Max: 2000}) // 過労
	world.Components.HealthStatus.Add(actor, &gc.HealthStatus{})

	progressTurnFatigue(world)

	cond := world.Components.HealthStatus.Get(actor).Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionExhaustion)
	require.NotNil(t, cond, "過労なら Exhaustion 不調が立つ")
	assert.Equal(t, gc.SeveritySevere, cond.Severity)
}

func TestProgressTurnFatigue_快調ならExhaustion不調は立たない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 0, Max: 2000})
	world.Components.HealthStatus.Add(actor, &gc.HealthStatus{})

	progressTurnFatigue(world)

	cond := world.Components.HealthStatus.Get(actor).Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionExhaustion)
	assert.Nil(t, cond, "快調なら不調は立たない")
}

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
