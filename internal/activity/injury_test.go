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
		assert.Positive(t, gc.BodyPartHitWeight(part), "重みのある部位だけ選ぶ")
	}
}

func TestApplyInjury(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 初回の判定が 15% 未満になる種で、1回の命中で確実に怪我を付ける。
	// spawn が乱数を消費しうるので判定の直前に種を固定する
	world.Resources.Config.RNG = rand.New(rand.NewPCG(4, 0))
	applyInjury(player, player, world, &gc.Melee{AttackCategory: gc.AttackSword})

	hs := world.Components.HealthStatus.Get(player)
	total, hit := 0, gc.BodyPartCount
	for p := range gc.BodyPartCount {
		if n := hs.Parts[p].CountConditions(gc.ConditionLaceration); n > 0 {
			total += n
			hit = p
		}
	}
	assert.Equal(t, 1, total, "1回で切り傷が1つ付く。斬撃なので切り傷")
	assert.NotEqual(t, gc.BodyPartWholeBody, hit, "全身は命中先にしない")
	assert.True(t, world.Components.StatsChanged.Has(player), "StatsChanged を立てる")
}

func TestApplyInjury_判定に外れると付かない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.HealthStatus.Add(entity, &gc.HealthStatus{})

	// 初回の判定が 15% 以上になる種では怪我が付かない
	world.Resources.Config.RNG = rand.New(rand.NewPCG(1, 0))
	applyInjury(entity, entity, world, &gc.Melee{AttackCategory: gc.AttackSword})

	hs := world.Components.HealthStatus.Get(entity)
	for p := range gc.BodyPartCount {
		assert.Empty(t, hs.Parts[p].Conditions, "判定に外れると怪我は付かない")
	}
	assert.False(t, world.Components.StatsChanged.Has(entity), "StatsChanged も立たない")
}

func TestApplyInjury_同種の怪我はソフト上限で頭打ち(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	hs := &gc.HealthStatus{}
	// 命中しうる全部位を切り傷で上限まで埋める。どの部位が抽選されても上限で弾かれる
	for p := range gc.BodyPartCount {
		if gc.BodyPartHitWeight(p) == 0 {
			continue
		}
		for range maxInjuriesPerPartType {
			hs.Parts[p].AddCondition(gc.HealthCondition{Type: gc.ConditionLaceration, Timer: injuryInitialTimer})
		}
	}
	world.Components.HealthStatus.Add(entity, hs)
	before := 0
	for p := range gc.BodyPartCount {
		before += hs.Parts[p].CountConditions(gc.ConditionLaceration)
	}

	// 判定は通る種でも、上限に達した部位には足さない
	world.Resources.Config.RNG = rand.New(rand.NewPCG(4, 0))
	applyInjury(entity, entity, world, &gc.Melee{AttackCategory: gc.AttackSword})

	after := 0
	for p := range gc.BodyPartCount {
		after += hs.Parts[p].CountConditions(gc.ConditionLaceration)
		assert.LessOrEqual(t, hs.Parts[p].CountConditions(gc.ConditionLaceration), maxInjuriesPerPartType, "上限を超えない")
	}
	assert.Equal(t, before, after, "上限に達した種類は増えない")
}

func TestApplyInjury_敵も同じ機構で怪我を負う(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 8, Y: 8}, "fireball")
	require.NoError(t, err)

	// 敵はプレイヤーと同じ共通土台で健康・効率機構を持つ
	require.True(t, world.Components.HealthStatus.Has(enemy), "敵も HealthStatus を持つ")
	require.True(t, world.Components.CharModifiers.Has(enemy), "敵も CharModifiers を持つ")

	// 1回の命中で確実に怪我を付ける
	world.Resources.Config.RNG = rand.New(rand.NewPCG(4, 0))
	applyInjury(enemy, enemy, world, &gc.Melee{AttackCategory: gc.AttackSword})

	hs := world.Components.HealthStatus.Get(enemy)
	total := 0
	for p := range gc.BodyPartCount {
		total += len(hs.Parts[p].Conditions)
	}
	assert.Equal(t, 1, total, "こちらの攻撃で敵にも怪我が付く")
	assert.True(t, world.Components.StatsChanged.Has(enemy), "敵も再計算を促す")
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
