package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// NewRefillMenuState は補充メニューを作る。持っている満たせる物のうち満タンでない物を並べ、
// 選ぶとその物をくべ先にした給油メニューを開く。満タンにして戻ると一覧から消える
func NewRefillMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(refillChoices), nil
}

func refillChoices(world w.World) (string, []Choice) {
	var choices []Choice
	for _, item := range refillableItems(world) {
		choices = append(choices, Choice{
			Label: query.GetEntityName(item, world),
			Run: pushChoice(func() (es.State[w.World], error) {
				return NewFeedFuelMenuState(LightFuelSink(item))
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

// refillableItems はプレイヤーが持つ物のうち、燃料で満たせて満タンでない物を返す
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
		if _, ok := query.RefillTagOf(world, e); !ok {
			continue
		}
		if c := world.Components.Charge.Get(e); c.Current < c.Max {
			items = append(items, e)
		}
	}
	return query.SortEntities(world, items)
}
