package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
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
	_, _, drainErr := gameaction.DrainContainer(world, light)
	choices := []Choice{
		{
			Label:    query.T(world, "Fill"),
			Disabled: charge.Current >= charge.Max || !carriesFuelFor(world, light),
			Run: pushChoice(func() (es.State[w.World], error) {
				return NewFeedFuelMenuState(LightFuelSink(world, light))
			}),
		},
		{
			Label:    query.T(world, "Drain"),
			Disabled: charge.Current <= 0 || drainErr != nil,
			Run: popAfter(func(world w.World) error {
				fuel, _, err := gameaction.DrainCharge(world, light)
				if err != nil {
					return err
				}
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "Drained %s into %s.",
						gamelog.Tag("item", query.GetEntityName(light, world)),
						gamelog.Tag("item", query.GetEntityName(fuel, world)))).
					Log()
				return nil
			}),
		},
		{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}},
	}
	return query.GetEntityName(light, world), choices
}

// carriesFuelFor はプレイヤーが灯りの light に合う燃料を持っているかを返す。給油メニューが並べる燃料と同じ条件で見る
func carriesFuelFor(world w.World, light ecs.Entity) bool {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}
	sink := LightFuelSink(world, light)
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
