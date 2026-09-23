package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/widgets/styled"
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

// fuelHeatCell は燃料メニューの熱量列のセルを返す。束の総熱量を炎アイコン付きで整形する。
// 熱量という燃料ドメインの知識をここに閉じ、汎用の収納メニューへ持ち込まない
func fuelHeatCell(world w.World, e ecs.Entity, count int) string {
	return (query.HeatContent(world, e) * consts.Heat(count)).String()
}

// NewCubeMenuState は移動拠点キューブの入口メニューを作る。直上で Enter すると開き、運転・展開と圧縮の
// 切替・燃料投入・キューブ情報の下位項目へ分岐する。燃料投入は可燃物だけを受け入れ、運転燃料に充てる。
// 運転は展開中にはできないので、圧縮中のときだけ項目に出す。
func NewCubeMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		choices := []Choice{deployChoice(world, cube)}
		if !world.Components.Deployed.Has(cube) {
			choices = append(choices, driveChoice(world, cube))
		}
		choices = append(choices,
			Choice{Label: query.T(world, "Fuel"), Run: pushChoice(func() (es.State[w.World], error) {
				// 見出しに残燃料を出し、投入するたび即座に増えるのを見せる。炎アイコンと数字で燃料と分かる
				fuelTitle := func(world w.World) string {
					return query.CubeFuelTotal(world, cube).String()
				}
				// 熱量を重量の左へ。右寄せ数値どうしが詰まらないよう空の間隔列を挟む
				emptyCell := func(w.World, ecs.Entity, int) string { return "" }
				return NewStorageMenuState(cube,
					WithItemFilter(isFuelItem),
					WithStoreOnly(),
					WithTitle(fuelTitle),
					WithColumn(styled.Num(), fuelHeatCell),
					WithColumn(styled.Fit(), emptyCell),
				)
			})},
			Choice{Label: query.T(world, "Cube info"), Run: pushChoice(func() (es.State[w.World], error) {
				return NewCubeInfoState(cube)
			})},
			Choice{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}},
		)
		return query.T(world, "Cube"), choices
	}), nil
}

// driveChoice は圧縮中のキューブに乗り込んで運転を始める項目。プレイヤーへ Driving を付けると、以後の
// 移動入力がキューブを動かす。展開中は呼び出し側が項目に出さない。運転中は DungeonState が入力を移動と
// 降車だけに絞りメニューを開けないので、Driving が既に付いた状態でここへ来ることはない。ゆえに二重付与の
// ガードは置かず、万一その不変条件が破れたら Ark の二重 Add で loud に露見させる。
func driveChoice(world w.World, cube ecs.Entity) Choice {
	return Choice{Label: query.T(world, "Drive"), Run: func(world w.World) (es.Transition[w.World], error) {
		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "You board the cube and start driving.")).
			Log()
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
		if !lifecycle.DeployCube(world, cube) {
			gamelog.New(query.GetGameLog(world)).
				Markup(query.T(world, "Not enough open space to deploy.")).
				Log()
		}
		// 成否によらずメニューは閉じる。失敗理由はログに出るので、場所を変えて開き直して再挑戦する
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}}
}
