package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionDefsInjury_攻撃種別ごとの怪我(t *testing.T) {
	t.Parallel()

	defs := testutil.InitTestWorld(t).Resources.ConditionDefs
	got := map[oapi.AttackCategory]gc.ConditionType{}
	for _, at := range []oapi.AttackCategory{oapi.AttackCategorySword, oapi.AttackCategorySpear, oapi.AttackCategoryHandgun, oapi.AttackCategoryRifle, oapi.AttackCategoryFist, oapi.AttackCategoryCannon, oapi.AttackCategoryBow} {
		ct, ok := defs.Injury(at)
		require.True(t, ok, at)
		got[at] = ct
	}
	// 打撃と砲撃は骨折、刃と弾は切り傷
	assert.Equal(t, map[oapi.AttackCategory]gc.ConditionType{
		oapi.AttackCategorySword:   gc.ConditionType("laceration"),
		oapi.AttackCategorySpear:   gc.ConditionType("laceration"),
		oapi.AttackCategoryHandgun: gc.ConditionType("laceration"),
		oapi.AttackCategoryRifle:   gc.ConditionType("laceration"),
		oapi.AttackCategoryFist:    gc.ConditionType("fracture"),
		oapi.AttackCategoryCannon:  gc.ConditionType("fracture"),
		oapi.AttackCategoryBow:     gc.ConditionType("laceration"),
	}, got)
}

type fixedRoll float64

func (f fixedRoll) Float64() float64 { return float64(f) }

func TestHitPart_起こりうる部位だけへ当たる(t *testing.T) {
	t.Parallel()

	// [0, 1) を細かく刻んで走査し、命中先の部位を出る順に固定する
	tests := []struct {
		ct   gc.ConditionType
		want []gc.BodyPart
	}{
		{gc.ConditionType("fracture"), []gc.BodyPart{gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet}},
		{gc.ConditionType("laceration"), []gc.BodyPart{gc.BodyPartHead, gc.BodyPartTorso, gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet}},
	}
	defs := testutil.InitTestWorld(t).Resources.ConditionDefs
	for _, tt := range tests {
		seen := map[gc.BodyPart]bool{}
		var got []gc.BodyPart
		const steps = 1000
		for k := range steps {
			part := hitPart(fixedRoll(float64(k)/steps), defs, tt.ct)
			if !seen[part] {
				seen[part] = true
				got = append(got, part)
			}
		}
		assert.Equal(t, tt.want, got, "%s の命中先", tt.ct)
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
	applyInjury(player, player, world, &gc.Melee{AttackCategory: oapi.AttackCategorySword})

	hs := world.Components.HealthStatus.Get(player)
	total, hit := 0, gc.BodyPartCount
	for p := range gc.BodyPartCount {
		if n := hs.Parts[p].CountConditions(gc.ConditionType("laceration")); n > 0 {
			total += n
			hit = p
		}
	}
	assert.Equal(t, 1, total, "1回で切り傷が1つ付く。斬撃なので切り傷")
	assert.NotEqual(t, gc.BodyPartWholeBody, hit, "全身は命中先にしない")
}

func TestApplyInjury_判定に外れると付かない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.HealthStatus.Add(entity, &gc.HealthStatus{})

	// 初回の判定が 15% 以上になる種では怪我が付かない
	world.Resources.Config.RNG = rand.New(rand.NewPCG(1, 0))
	applyInjury(entity, entity, world, &gc.Melee{AttackCategory: oapi.AttackCategorySword})

	hs := world.Components.HealthStatus.Get(entity)
	for p := range gc.BodyPartCount {
		assert.Empty(t, hs.Parts[p].Conditions, "判定に外れると怪我は付かない")
	}
}

func TestApplyInjury_同種の怪我は1部位に1つで頭打ち(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	hs := &gc.HealthStatus{}
	// 命中しうる全部位を切り傷で上限まで埋める。どの部位が抽選されても上限で弾かれる
	for p := range gc.BodyPartCount {
		if world.Resources.ConditionDefs.HitWeight(p) == 0 {
			continue
		}
		for range maxInjuriesPerPartType {
			hs.Parts[p].AddCondition(gc.HealthCondition{Type: gc.ConditionType("laceration"), Timer: injuryInitialTimer})
		}
	}
	world.Components.HealthStatus.Add(entity, hs)
	before := 0
	for p := range gc.BodyPartCount {
		before += hs.Parts[p].CountConditions(gc.ConditionType("laceration"))
	}

	// 判定は通る種でも、上限に達した部位には足さない
	world.Resources.Config.RNG = rand.New(rand.NewPCG(4, 0))
	applyInjury(entity, entity, world, &gc.Melee{AttackCategory: oapi.AttackCategorySword})

	after := 0
	for p := range gc.BodyPartCount {
		after += hs.Parts[p].CountConditions(gc.ConditionType("laceration"))
		assert.LessOrEqual(t, hs.Parts[p].CountConditions(gc.ConditionType("laceration")), maxInjuriesPerPartType, "上限を超えない")
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

	// 1回の命中で確実に怪我を付ける
	world.Resources.Config.RNG = rand.New(rand.NewPCG(4, 0))
	applyInjury(enemy, enemy, world, &gc.Melee{AttackCategory: oapi.AttackCategorySword})

	hs := world.Components.HealthStatus.Get(enemy)
	total := 0
	for p := range gc.BodyPartCount {
		total += len(hs.Parts[p].Conditions)
	}
	assert.Equal(t, 1, total, "こちらの攻撃で敵にも怪我が付く")
}

func TestApplyInjury_HealthStatusなしは何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.RNG = rand.New(rand.NewPCG(7, 0))
	entity := world.ECS.NewEntity() // HealthStatus を持たない裸エンティティ
	attack := &gc.Melee{AttackCategory: oapi.AttackCategorySword}

	applyInjury(entity, entity, world, attack) // panic せず何も起きない
}
