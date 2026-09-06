package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

// setExhaustion は疲労システムが立てるのと同じ Exhaustion 不調を WholeBody へ立てる。
// 単体テストはシステムを回さないので、同期後の状態を直接作る
func setExhaustion(world w.World, entity ecs.Entity, sev gc.Severity) {
	hs := &gc.HealthStatus{}
	hs.Parts[gc.BodyPartWholeBody].SetGaugeCondition(gc.ConditionExhaustion, sev)
	world.Components.HealthStatus.Add(entity, hs)
}

func TestProficiencyValue_疲労は武器命中を下げる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())

	base := int(ProficiencyValue(world, entity, gc.ProfSwordAccuracy))

	// 過労の Exhaustion 不調は全身性 10*3=30 を意識へ落とし、操作機能70%が命中へ乗る
	setExhaustion(world, entity, gc.SeveritySevere)
	tired := int(ProficiencyValue(world, entity, gc.ProfSwordAccuracy))

	assert.Equal(t, base*70/100, tired, "過労で命中が70%に下がる")
}

func TestProficiencyValue_疲労は命中以外に効かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())
	setExhaustion(world, entity, gc.SeveritySevere)

	// ProfMaxWeight は身体機能を畳まないので、全身性の低下では変わらない
	assert.Equal(t, int(gc.CalcProficiencyValue(gc.NewSkills(), nil, gc.HealthyBodyFuncs(), gc.ProfMaxWeight)),
		int(ProficiencyValue(world, entity, gc.ProfMaxWeight)), "身体機能を畳まない倍率キーは全身性で変わらない")
}

func TestProficiencySources_疲労の内訳が値と一致する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())
	setExhaustion(world, entity, gc.SeveritySevere)

	value := int(ProficiencyValue(world, entity, gc.ProfSwordAccuracy))
	sum := int(consts.PercentBase)
	var sawBodyFunc bool
	for _, s := range ProficiencySources(world, entity, gc.ProfSwordAccuracy) {
		sum += s.Value
		// 疲労は意識を下げ、命中は操作機能の畳み込みとして内訳に載る
		if s.Kind == gc.SourceBodyFunc {
			sawBodyFunc = true
		}
	}
	assert.Equal(t, value, sum, "内訳の合計は値に一致する")
	assert.True(t, sawBodyFunc, "疲労が下げた身体機能の内訳が載る")
}
