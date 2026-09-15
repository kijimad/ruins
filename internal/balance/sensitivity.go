package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/query"
)

// Knob は分析機械が動かすパラメータベクトルの1成分。Ptr が Params 内の成分を指すので、
// 感度・交換レート・探索は成分の意味を知らずに摂動できる。
type Knob struct {
	Name   string
	Domain string
	Ptr    func(*Params) *float64
}

// knobRegistry は分析機械が回すつまみの一覧。つまみを増やすときはここに1行足すだけでよい。
// Params に成分がある値なら導出・感度・交換レートへ自動で載る。
func knobRegistry() []Knob {
	return []Knob{
		{"敵武器ダメージ(bite)", "戦闘", func(p *Params) *float64 { return &p.EnemyWeaponScale }},
		{"プレイヤー筋力", "戦闘", func(p *Params) *float64 { return &p.PlayerStrengthScale }},
		{"最大満腹度", "生存", func(p *Params) *float64 { return &p.MaxHunger }},
		{"空腹減耗ターン", "生存", func(p *Params) *float64 { return &p.HungerDrainTurns }},
		{"飢餓しきい値", "生存", func(p *Params) *float64 { return &p.HungerStarvingRatio }},
		{"疲労蓄積量", "疲労", func(p *Params) *float64 { return &p.FatigueGainPerTurn }},
		{"疲労回復量", "疲労", func(p *Params) *float64 { return &p.FatigueRecoverPerTurn }},
		{"燃料熱量", "物流", func(p *Params) *float64 { return &p.FuelHeatScale }},
		{"loot価値", "経済", func(p *Params) *float64 { return &p.LootValueScale }},
		{"1日ターン数", "横断", func(p *Params) *float64 { return &p.TurnsPerDay }},
	}
}

// SensitivityMetricNames は感度行列の列。ドメイン横断の代表メトリクス。
var SensitivityMetricNames = []string{"戦力比d20", "飢餓まで日数", "睡眠時間割合", "OIL航続", "loot手取り", "Lv30攻撃数"}

// metricValue はパラメータベクトル p の下で idx 番目のメトリクスだけを評価して返す。
// 導出はすべて p の純関数なので、成分を摂動すれば任意のつまみの影響を測れる。単一メトリクス
// 単位にするのは、交換レートの求解が対象メトリクス以外の重い計算を回さないようにするため。
// idx が範囲外なら 0 を返す。
func metricValue(master oapi.Raws, p Params, idx int) float64 {
	switch idx {
	case 0: // 戦闘。敵武器ダメージとプレイヤー筋力の倍率を適用して廃墟 day20 戦力比を測る
		player, err := LoadCombatantFromMember(master, BaselinePlayer)
		if err != nil {
			return 0
		}
		weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
		if err != nil {
			return 0
		}
		player.Strength = int(math.Round(float64(player.Strength) * p.PlayerStrengthScale))
		ratio := 0.0
		withScaledMeleeDamage(master, "bite", p.EnemyWeaponScale, func() {
			if curve, e := DifficultyCurve(master, player, weapon, BaselineAreaTable, 20); e == nil && len(curve) >= 20 {
				ratio = curve[19].PowerRatio
			}
		})
		return ratio
	case 1: // 生存
		return DaysUntilStarving(p)
	case 2: // 疲労
		return SleepTimeFraction(p)
	case 3: // 物流。燃料熱量の倍率を掛けて満載 OIL の航続を測る
		capacity := consts.Milligram(consts.CubeWeightCapacityKg) * consts.MilligramPerKg
		fuel := float64(query.HeatOf(oapi.OIL, capacity)) * p.FuelHeatScale
		return DriveRangeTiles(consts.Heat(fuel), capacity)
	case 4: // 経済。loot 価値の倍率を掛けて1個あたり手取りを測る。送料が固定なので弾力性は1を超える
		lv := ExpectedLootValue(master, "ruins_area", 8) * p.LootValueScale
		if lv <= 0 {
			return 0
		}
		return float64(query.AuctionNetProceeds(consts.Currency(math.Round(lv)), ExpectedLootWeightKg(master, "ruins_area", 8)))
	case 5: // 進行。現状の成分では動かない。他ドメインのつまみが成長へ波及しないことを見せる列
		return float64(AttacksToSkillLevel(0, 30))
	}
	return 0
}

// metricsAt はパラメータベクトル p の下で全メトリクスを評価して返す。感度行列が全列を要するので
// metricValue を全 idx について回す。
func metricsAt(master oapi.Raws, p Params) []float64 {
	out := make([]float64, len(SensitivityMetricNames))
	for i := range out {
		out[i] = metricValue(master, p, i)
	}
	return out
}

// SensitivityCell は1つのメトリクスへの、つまみ+10%あたりの変化率。絶対値が大きいほど強い結合。
type SensitivityCell struct {
	Metric    string
	PctChange float64 // (摂動値-現状)/現状 の百分率。つまみ+10%あたり
}

// KnobSensitivity は1つのつまみを+10%したときの、各メトリクスへの影響。
type KnobSensitivity struct {
	Knob   string
	Domain string
	Cells  []SensitivityCell
}

// CrossDomainSensitivity はドメイン横断の感度行列を返す。knobRegistry の各つまみを+10%し、
// 全ドメインの代表メトリクスがどれだけ動くかを測る。ヤコビアンで、多くはブロック対角、
// すなわち各つまみは自分のドメインのメトリクスだけを動かす。横断するのは1日ターン数の
// ような共通の分母だけ。
func CrossDomainSensitivity(master oapi.Raws) []KnobSensitivity {
	base := metricsAt(master, DefaultParams())
	knobs := knobRegistry()
	rows := make([]KnobSensitivity, 0, len(knobs))
	for _, k := range knobs {
		p := DefaultParams()
		*k.Ptr(&p) *= 1.1
		pert := metricsAt(master, p)
		cells := make([]SensitivityCell, len(SensitivityMetricNames))
		for i := range SensitivityMetricNames {
			pct := 0.0
			if base[i] != 0 {
				pct = (pert[i] - base[i]) / base[i] * 100
			}
			cells[i] = SensitivityCell{Metric: SensitivityMetricNames[i], PctChange: pct}
		}
		rows = append(rows, KnobSensitivity{Knob: k.Name, Domain: k.Domain, Cells: cells})
	}
	return rows
}
