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

// NewCubeMenuState は移動拠点キューブの入口メニューを作る。隣接で開き、随伴の命令・展開と圧縮の
// 切替・給油・キューブ情報の下位項目へ分岐する。給油は持ち物の可燃物を燃料ポイントへくべる。
// 状態で使えない項目は消さずに無効表示で残す。追従は展開中、設備は圧縮中に無効になる。
func NewCubeMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		deployed := world.Components.Deployed.Has(cube)
		choices := []Choice{
			deployChoice(world, cube),
			followChoice(world, cube, deployed),
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
			Choice{Label: query.T(world, "Cube info"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewCubeInfoState(cube)
			})},
			Choice{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}},
		}
		return query.T(world, "Cube"), choices
	}), nil
}

// followChoice は随伴の命令を切り替える項目を返す。待機中なら追従を、追従中なら待機を命じる。
// 追従は1マスの圧縮形態でだけできるので、展開中は無効表示にして消さない。
func followChoice(world w.World, cube ecs.Entity, deployed bool) Choice {
	solo := world.Components.SoloAI.Get(cube)
	if solo.Movement == gc.SoloFollow {
		return Choice{Label: query.T(world, "Stay here"), Run: func(world w.World) (es.Transition[w.World], error) {
			world.Components.SoloAI.Get(cube).Movement = gc.SoloStationary
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}}
	}
	return Choice{Label: query.T(world, "Follow me"), Disabled: deployed, Run: func(world w.World) (es.Transition[w.World], error) {
		world.Components.SoloAI.Get(cube).Movement = gc.SoloFollow
		return es.Transition[w.World]{Type: es.TransPop}, nil
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
		// 展開中は随伴しないので、展開に成功したら待機させる
		if lifecycle.DeployCube(world, cube) {
			world.Components.SoloAI.Get(cube).Movement = gc.SoloStationary
		} else {
			gamelog.New(query.GetGameLog(world)).
				Markup(query.T(world, "Not enough open space to deploy.")).
				Log()
		}
		// 成否によらずメニューは閉じる。失敗理由はログに出るので、場所を変えて開き直して再挑戦する
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}}
}
