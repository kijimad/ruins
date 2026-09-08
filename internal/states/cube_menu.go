package states

import (
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// NewCubeMenuState は移動拠点キューブの入口メニューを作る。隣接時に開き、収納・オークション・
// キューブ情報の下位項目へ分岐する。乗車は直上 Enter で完結するのでここには並べない。
func NewCubeMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		return query.T(world, "Cube"), []Choice{
			{Label: query.T(world, "Storage"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewStorageMenuState(cube)
			})},
			{Label: query.T(world, "Auction"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewAuctionMenuState(cube)
			})},
			{Label: query.T(world, "Cube info"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewCubeInfoState(cube)
			})},
			{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}},
		}
	}), nil
}
