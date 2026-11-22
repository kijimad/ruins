package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// NewRefillMenuState は補充メニューを作る。持っている灯りを選び、満たすか抜くかを選ぶ
func NewRefillMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(refillChoices), nil
}

func refillChoices(world w.World) (string, []Choice) {
	var choices []Choice
	for _, light := range refillableItems(world) {
		choices = append(choices, Choice{
			Label: query.GetEntityName(light, world),
			Value: consts.Turn(world.Components.Charge.Get(light).Current).String(),
			Run: pushChoice(func() (es.State[w.World], error) {
				return NewChoiceMenu(func(world w.World) (string, []Choice) { return lightRefillChoices(world, light) }), nil
			}),
		})
	}
	if len(choices) == 0 {
		choices = append(choices, Choice{Label: query.T(world, "Nothing to refill"), Disabled: true})
	}
	choices = append(choices, Choice{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}})
	return query.T(world, "Refill"), choices
}

// lightRefillChoices は灯りの light を満たすか抜くかの選択肢を返す。使えない方は選べない行にする
func lightRefillChoices(world w.World, light ecs.Entity) (string, []Choice) {
	charge := world.Components.Charge.Get(light)
	fill := LightFuelSink(world, light)
	drain := LightDrainSink(world, light)
	choices := []Choice{
		{
			Label:    query.T(world, "Fill"),
			Disabled: charge.Current >= charge.Max || !carriesAccepted(world, fill),
			Run: pushChoice(func() (es.State[w.World], error) {
				return NewFeedFuelMenuState(fill)
			}),
		},
		{
			Label:    query.T(world, "Drain"),
			Disabled: charge.Current <= 0 || !carriesAccepted(world, drain),
			Run: pushChoice(func() (es.State[w.World], error) {
				return NewFeedFuelMenuState(drain)
			}),
		},
		{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}},
	}
	return query.GetEntityName(light, world), choices
}

// carriesAccepted はプレイヤーの持ち物に sink が受ける物があるかを返す
func carriesAccepted(world w.World, sink FuelSink) bool {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}
	for _, stack := range query.BackpackStacks(world, player) {
		if sink.accepts(world, stack.Rep) {
			return true
		}
	}
	return false
}

// refillableItems はプレイヤーが持つ物のうち、燃料を受ける灯りを満タンの物も含めて返す
func refillableItems(world w.World) []ecs.Entity {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil
	}
	var items []ecs.Entity
	q := query.ActiveFilter1[gc.Charge](world).Query()
	for q.Next() {
		e := q.Entity()
		if !query.CarriedBy(world, e, player) {
			continue
		}
		if _, ok := query.AcceptedFuelOf(world, e); ok {
			items = append(items, e)
		}
	}
	return query.SortEntities(world, items)
}
