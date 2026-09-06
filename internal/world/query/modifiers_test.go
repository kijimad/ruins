package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProficiencyValue_疲労は武器命中を下げる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())

	base := int(ProficiencyValue(world, entity, gc.ProfSwordAccuracy))

	// 過労を付与すると命中が ×75% になる
	world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})
	tired := int(ProficiencyValue(world, entity, gc.ProfSwordAccuracy))

	assert.Equal(t, base*75/100, tired, "過労で命中が75%に下がる")
}

func TestProficiencyValue_疲労は命中以外に効かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())
	world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})

	// ProfMaxWeight は身体機能を畳まないので、疲労の意識低下では変わらない
	assert.Equal(t, int(gc.CalcProficiencyValue(gc.NewSkills(), nil, gc.HealthyBodyFunctions(), gc.ProfMaxWeight)),
		int(ProficiencyValue(world, entity, gc.ProfMaxWeight)), "身体機能を畳まない倍率キーは疲労で変わらない")
}

func TestProficiencySources_疲労の内訳が値と一致する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())
	world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})

	value := int(ProficiencyValue(world, entity, gc.ProfSwordAccuracy))
	sum := int(consts.PercentBase)
	var sawBodyFunction bool
	for _, s := range ProficiencySources(world, entity, gc.ProfSwordAccuracy) {
		sum += s.Value
		// 疲労は意識を下げ、命中は操作機能の畳み込みとして内訳に載る
		if s.Kind == gc.SourceBodyFunction {
			sawBodyFunction = true
		}
	}
	assert.Equal(t, value, sum, "内訳の合計は値に一致する")
	assert.True(t, sawBodyFunction, "疲労が下げた身体機能の内訳が載る")
}
