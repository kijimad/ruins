package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestModifierValue_疲労は武器命中を下げる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())

	base := int(ModifierValue(world, entity, gc.ModSwordAccuracy))

	// 過労を付与すると命中が ×75% になる
	world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})
	tired := int(ModifierValue(world, entity, gc.ModSwordAccuracy))

	assert.Equal(t, base*75/100, tired, "過労で命中が75%に下がる")
}

func TestModifierValue_疲労は命中以外に効かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())
	world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})

	// 命中でないキーは疲労の影響を受けない
	assert.Equal(t, int(gc.CalcModifierValue(gc.NewSkills(), nil, nil, gc.ModMaxWeight)),
		int(ModifierValue(world, entity, gc.ModMaxWeight)), "命中でない倍率キーは疲労で変わらない")
}

func TestModifierSources_疲労の内訳が値と一致する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	world.Components.Skills.Add(entity, gc.NewSkills())
	world.Components.Fatigue.Add(entity, &gc.Fatigue{Current: 900, Max: 1000})

	value := int(ModifierValue(world, entity, gc.ModSwordAccuracy))
	sum := int(consts.PercentBase)
	var sawFatigue bool
	for _, s := range ModifierSources(world, entity, gc.ModSwordAccuracy) {
		sum += s.Value
		if s.Kind == gc.SourceFatigue {
			sawFatigue = true
		}
	}
	assert.Equal(t, value, sum, "内訳の合計は値に一致する")
	assert.True(t, sawFatigue, "疲労の内訳が載る")
}
