package states

import (
	"fmt"

	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// NewCubeModuleMenuState はキューブのモジュール装着メニューを作る。装着中のモジュールを外し、
// バックパックのモジュールを装着する。展開範囲は装着で即変わるので見出しに現在のフットプリントを出す。
func NewCubeModuleMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		title := cubeModuleMenuTitle(world, cube)

		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return title, []Choice{closeChoice(world)}
		}

		var choices []Choice

		installed := query.GetCubeModules(world, cube)
		if len(installed) > 0 {
			choices = append(choices, Choice{Label: query.T(world, "Installed"), Header: true})
			for _, m := range installed {
				choices = append(choices, Choice{
					Label:  query.GetEntityName(m, world),
					Value:  query.T(world, "Remove"),
					Indent: 1,
					Run: stayAfter(func(world w.World) error {
						return lifecycle.MoveToStorage(world, m, cube)
					}),
				})
			}
		}

		// 装着候補はキューブ収納とプレイヤーのバックパックの両方から集める。取り外したモジュールは
		// 収納へ戻すので、外して付け替える運用が1画面で完結する
		available := append(query.StorageCubeModules(world, cube), query.BackpackCubeModules(world, player)...)
		if len(available) > 0 {
			choices = append(choices, Choice{Label: query.T(world, "Attach"), Header: true})
			for _, m := range available {
				choices = append(choices, Choice{
					Label:  query.GetEntityName(m, world),
					Value:  query.T(world, "Install"),
					Indent: 1,
					Run: stayAfter(func(world w.World) error {
						lifecycle.MoveToCubeModule(world, m, cube)
						return nil
					}),
				})
			}
		}

		choices = append(choices, closeChoice(world))
		return title, choices
	}), nil
}

// cubeModuleMenuTitle は展開野営のフットプリントを WxH タイルで見出しにする。半径は縦横別で、
// フットプリントは中心を含むので 2*半径+1 になる。装着で範囲が伸びるのを見出しで確かめられる。
func cubeModuleMenuTitle(world w.World, cube ecs.Entity) string {
	r := query.CubeDeployRange(world, cube)
	return fmt.Sprintf("%s %dx%d", query.T(world, "Deploy range"), 2*r.X+1, 2*r.Y+1)
}

// closeChoice は閉じる項目を返す。モジュールメニューの末尾に置く。
func closeChoice(world w.World) Choice {
	return Choice{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}}
}
