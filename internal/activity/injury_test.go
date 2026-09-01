package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjuryTypeFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, gc.ConditionFracture, injuryTypeFor(gc.AttackFist), "打撃は骨折")
	assert.Equal(t, gc.ConditionFracture, injuryTypeFor(gc.AttackCanon), "砲撃は骨折")
	assert.Equal(t, gc.ConditionLaceration, injuryTypeFor(gc.AttackSword), "斬撃は切り傷")
	assert.Equal(t, gc.ConditionLaceration, injuryTypeFor(gc.AttackRifle), "射撃は切り傷")
}

func TestRandomHitPart(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.RNG = rand.New(rand.NewPCG(1, 0))
	for range 200 {
		part := randomHitPart(world)
		assert.NotEqual(t, gc.BodyPartWholeBody, part, "全身は命中先にしない")
		assert.Positive(t, bodyPartHitWeights[part], "重みのある部位だけ選ぶ")
	}
}

func TestApplyInjury(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.RNG = rand.New(rand.NewPCG(7, 0))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	attack := &gc.Melee{AttackCategory: gc.AttackSword}

	// 15%を何度も回せば怪我は付く。独立に積むので合計が増える
	for range 200 {
		applyInjury(player, player, world, attack)
	}

	hs := world.Components.HealthStatus.Get(player)
	total := 0
	for p := range gc.BodyPartCount {
		total += len(hs.Parts[p].Conditions)
		if p == gc.BodyPartWholeBody {
			assert.Empty(t, hs.Parts[p].Conditions, "全身には怪我を付けない")
		}
		assert.LessOrEqual(t, hs.Parts[p].CountConditions(gc.ConditionLaceration), maxInjuriesPerPartType, "ソフト上限を超えない")
	}
	assert.Positive(t, total, "怪我が付く")
	assert.True(t, world.Components.StatsChanged.Has(player), "StatsChanged を立てる")
}

func TestApplyInjury_HealthStatusなしは何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.RNG = rand.New(rand.NewPCG(7, 0))
	entity := world.ECS.NewEntity() // HealthStatus を持たない裸エンティティ
	attack := &gc.Melee{AttackCategory: gc.AttackSword}

	applyInjury(entity, entity, world, attack) // panic せず何も起きない
	assert.False(t, world.Components.StatsChanged.Has(entity))
}
