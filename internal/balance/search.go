package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// ruinsDay20PowerRatio は現行 master の廃墟 day20 の戦力比を返す。探索と感度の目的関数。
func ruinsDay20PowerRatio(master oapi.Raws) float64 {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return 0
	}
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	if err != nil {
		return 0
	}
	curve, err := DifficultyCurve(master, player, weapon, BaselineAreaTable, BaselineDays)
	if err != nil || len(curve) == 0 {
		return 0
	}
	return curve[len(curve)-1].PowerRatio
}

// withScaledMeleeDamage は itemID の近接ダメージを factor 倍した状態で fn を呼び、呼び出し後に元へ戻す。
// master の item は backing array を共有するので、値を退避・復元して master を汚さずに評価する。
// 同一 master を並行して変更すると退避・復元が競合する。並行で使うときは master を共有せず、
// 呼び出しごとに独立した master をロードすること。
func withScaledMeleeDamage(master oapi.Raws, itemID string, factor float64, fn func()) {
	items := raw.PtrSlice(master.Items)
	for i := range items {
		if items[i].Id == itemID && items[i].Melee != nil {
			orig := items[i].Melee.Damage
			// 丸めで反映する。切り捨てだと factor=1.1 で奇数値が変化せず摂動が無反応になる
			items[i].Melee.Damage = int(math.Round(float64(orig) * factor))
			fn()
			items[i].Melee.Damage = orig
			return
		}
	}
	fn()
}

// SolveScalar は eval が単調な区間 [lo,hi] で eval(x)=target となる x を二分探索で返す。
// 端点が target を挟まなければ、その区間では両立不能として ok=false を返す。
// iters は正の反復回数を渡す。0以下だと精緻化せず区間中点を返す。
func SolveScalar(eval func(float64) float64, target, lo, hi float64, iters int) (float64, bool) {
	flo, fhi := eval(lo), eval(hi)
	if (flo-target)*(fhi-target) > 0 {
		return 0, false
	}
	for range iters {
		mid := (lo + hi) / 2
		fm := eval(mid)
		if (flo-target)*(fm-target) <= 0 {
			hi = mid
		} else {
			lo, flo = mid, fm
		}
	}
	return (lo + hi) / 2, true
}

// SensitivityRow は1つのパラメータを動かしたときの day20 戦力比の応答。
type SensitivityRow struct {
	Knob       string  // 動かしたパラメータ
	Base       float64 // 現状の day20 戦力比
	Plus10     float64 // そのパラメータを+10%したときの day20 戦力比
	DeltaRatio float64 // 変化率。(Plus10-Base)/Base
}

// sensitivityKnobs は感度表に載せる近接武器アイテム。序盤〜中盤の戦力比に効く敵武器とプレイヤー武器。
var sensitivityKnobs = []string{"bite", "cleaver", "flame_attack", "bare_hands"}

// SensitivityDay20 は各つまみを+10%したときの廃墟 day20 戦力比の応答を返す。
// どのパラメータがどれだけ効くかを定量化する。感度は関係グラフの矢印の重みに対応する。
func SensitivityDay20(master oapi.Raws) []SensitivityRow {
	base := ruinsDay20PowerRatio(master)
	rows := make([]SensitivityRow, 0, len(sensitivityKnobs))
	for _, knob := range sensitivityKnobs {
		var plus float64
		withScaledMeleeDamage(master, knob, 1.1, func() { plus = ruinsDay20PowerRatio(master) })
		delta := 0.0
		if base != 0 {
			delta = (plus - base) / base
		}
		rows = append(rows, SensitivityRow{Knob: knob, Base: base, Plus10: plus, DeltaRatio: delta})
	}
	return rows
}
