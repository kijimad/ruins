package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// materialHeatPerKg は材質のkgあたり燃焼熱量。ここに載らない材質は不燃で0とみなし燃料にならない。
// 燃料熱量は重量kgへ掛けて導く。石炭・油が高く、食料・骨が低い。金属・石・ガラス・結晶・陶磁器・液体は不燃。
// 係数は balance 値なので schema でなくここで持つ。地面効率50%で hardwood 3kg が約300ターン、
// coal 1kg が約400ターン燃えるよう釣り合わせる。火を絶やさぬ手触りに合わせて全体をここで振る
var materialHeatPerKg = map[oapi.Material]int{
	oapi.OIL:     1000,
	oapi.COAL:    800,
	oapi.PAPER:   300,
	oapi.PLASTIC: 240,
	oapi.WOOD:    200,
	oapi.CLOTH:   160,
	oapi.LEATHER: 120,
	oapi.PLANT:   120,
	oapi.FOOD:    60,
	oapi.BONE:    40,
}

// HeatOf は材質と重量から燃焼熱量を導く純関数。不燃の材質は0を返す。
// 整数で閉じるため mg 基準で計算し 1kg=1_000_000mg で割って kg 換算する。切り捨てで軽い物は0になる
func HeatOf(material oapi.Material, weight consts.Milligram) consts.Heat {
	perKg, ok := materialHeatPerKg[material]
	if !ok {
		return 0
	}
	return consts.Heat(perKg * int(weight) / consts.MilligramPerKg)
}

// HeatContent はエンティティの燃焼熱量を Material と Weight から導く。
// 値を保持せず読み取り時に算出する。どちらかの component を欠けば0
func HeatContent(world w.World, entity ecs.Entity) consts.Heat {
	if !world.Components.Material.Has(entity) || !world.Components.Weight.Has(entity) {
		return 0
	}
	kind := world.Components.Material.Get(entity).Kind
	weight := world.Components.Weight.Get(entity).Milligram
	return HeatOf(kind, weight)
}

// IsCombustible は燃やせるか、すなわち燃焼熱量が正かを返す。可燃マーカーの代わりに使う
func IsCombustible(world w.World, entity ecs.Entity) bool {
	return HeatContent(world, entity) > 0
}
