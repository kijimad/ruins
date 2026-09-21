package states

import (
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// isFuelItem は燃料タンクへ入れられる品かを返す。運転で消費するのと同じ燃焼熱量、すなわち燃料性能を
// 持つ物だけ受け入れる。CubeFuelTotal/ConsumeCubeFuel と同じ HeatContent で判定し、メニューに出るのに
// 燃料にならないズレを防ぐ。畳み込んだ貨物は別ロケーション LocationStowed なので燃料タンクには現れない。
func isFuelItem(world w.World, e ecs.Entity) bool {
	return query.HeatContent(world, e) > 0
}

// NewCubeMenuState は移動拠点キューブの入口メニューを作る。隣接時に開き、展開と圧縮の切替と、
// 燃料投入・キューブ情報の下位項目へ分岐する。燃料投入は可燃物だけを受け入れ、運転燃料に充てる。
// 乗車は直上 Enter で完結するのでここには並べない。
func NewCubeMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		return query.T(world, "Cube"), []Choice{
			deployChoice(world, cube),
			{Label: query.T(world, "Fuel"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewStorageMenuState(cube, WithItemFilter(isFuelItem))
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
