package query

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FormatTurns はターン数へ時計アイコンを添える。燃焼の残ターンや給油で増えるターンの表示に使い、
// 数字だけで読みにくいのを防ぐ。単位をここに集約し表記を揃える。
// 通貨表記と同じく記号を先頭にし半角スペースで区切る
func FormatTurns(turns consts.Turn) string {
	return fmt.Sprintf("%s %d", consts.IconClock, int(turns))
}

// FormatTurnsDelta は給油で増えるターン数を符号付きで整形する。記号は先頭、符号は数字側に付け、
// くべると何ターン延びるかを示す。給油メニューの右寄せ列に使う
func FormatTurnsDelta(turns consts.Turn) string {
	return fmt.Sprintf("%s +%d", consts.IconClock, int(turns))
}

// groundBurnEfficiency は地面直の火の燃焼効率をパーセント。100 が等倍で、地面直は低効率で薪をすぐ食う。
// 効率は熱の強さでなく燃焼時間に効く。良い火の見返りは暖かさでなく薪の節約になる
const groundBurnEfficiency = 50

// BurnEfficiency は火が燃えている場所の燃焼効率をパーセントで返す。
// 今は地面直だけなので定数。将来かまどを足すときは Hearth の効率をここで分ける。
// 燃料を燃やし始めるたびにこの値を引くので、後からかまどを足しても同じ機構に載る
func BurnEfficiency(_ w.World, _ ecs.Entity) int {
	return groundBurnEfficiency
}

// EstimateBurnTurns は火があと何ターン燃えるかを返す。
// 火は燃料を貯めず残量だけを持ち、残量は毎ターン1ずつ減るので残量がそのままターン数になる。
// 燃えていなければ0
func EstimateBurnTurns(world w.World, fire ecs.Entity) consts.Turn {
	if !world.Components.Burning.Has(fire) {
		return 0
	}
	return world.Components.Burning.Get(fire).Remaining
}

// FuelBurnTurns は fuel を fire へくべたとき増える燃焼ターン数を返す。
// 熱量を火の場所の効率で割り引いた値で、給油メニューの「+Nターン」表示に使う
func FuelBurnTurns(world w.World, fire ecs.Entity, fuel ecs.Entity) consts.Turn {
	return HeatContent(world, fuel).BurnTurns(BurnEfficiency(world, fire))
}
