package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// NewCubeMenuState は移動拠点キューブの入口メニューを作る。状態で使えない項目は消さずに無効表示で残す。
func NewCubeMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		deployed := world.Components.Deployed.Has(cube)
		choices := []Choice{
			deployChoice(world, cube),
			followChoice(world, cube),
			Choice{Label: query.T(world, "Fuel"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewFeedFuelMenuState(CubeFuelSink(cube))
			})},
			Choice{Label: query.T(world, "Module"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewCubeModuleMenuState(cube)
			})},
			// 設備は展開空間のマスへ装着するので、圧縮中は無効表示にする
			Choice{Label: query.T(world, "Facility"), Disabled: !deployed, Run: pushChoice(func() (es.State[w.World], error) {
				return NewCubeFacilityMenuState(cube)
			})},
			Choice{Label: query.T(world, "Info"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewCubeInfoState(cube)
			})},
			Choice{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}},
		}
		return query.T(world, "Cube"), choices
	}), nil
}

// followChoice は随伴の方針を追従と待機で切り替える項目を返す。設定の切り替えなのでメニューは閉じない。
func followChoice(world w.World, cube ecs.Entity) Choice {
	solo := world.Components.SoloAI.Get(cube)
	if solo.Movement == gc.SoloFollow {
		return Choice{Label: query.T(world, "Stay here"), Run: func(world w.World) (es.Transition[w.World], error) {
			world.Components.SoloAI.Get(cube).Movement = gc.SoloStationary
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}}
	}
	return Choice{Label: query.T(world, "Follow me"), Run: func(world w.World) (es.Transition[w.World], error) {
		world.Components.SoloAI.Get(cube).Movement = gc.SoloFollow
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}}
}

// deployChoice は展開中なら圧縮、圧縮中なら展開の項目を返す。展開は必要な空きが無ければ拒否してログを出す。
func deployChoice(world w.World, cube ecs.Entity) Choice {
	if world.Components.Deployed.Has(cube) {
		return Choice{Label: query.T(world, "Compress"), Run: func(world w.World) (es.Transition[w.World], error) {
			lifecycle.StowCube(world, cube)
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}}
	}
	return Choice{Label: query.T(world, "Deploy"), Run: func(world w.World) (es.Transition[w.World], error) {
		if !lifecycle.DeployCube(world, cube) {
			gamelog.New(query.GetGameLog(world)).
				Markup(query.T(world, "Not enough open space to deploy.")).
				Log()
		}
		// 成否によらずメニューは閉じる。失敗理由はログに出るので、場所を変えて開き直して再挑戦する
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}}
}
