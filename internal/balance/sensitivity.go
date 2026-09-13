package balance

import (
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/world/query"
)

// sensScales は各つまみの倍率。1.0 が現状。感度分析はどれか1つを 1.1 にして +10% の影響を測る。
type sensScales struct {
	weapon, playerStr, maxHunger, hungerDrain, fatigueRecover, fuelHeat, lootValue, turnsPerDay float64
}

func baseScales() sensScales {
	return sensScales{1, 1, 1, 1, 1, 1, 1, 1}
}

// SensitivityMetricNames は感度行列の列。ドメイン横断の代表メトリクス。
var SensitivityMetricNames = []string{"戦力比d20", "飢餓まで日数", "睡眠時間割合", "OIL航続", "loot手取り", "Lv30攻撃数"}

// sensMetrics は倍率 s の下で SensitivityMetricNames の各メトリクスを評価して返す。単一出典の導出式を
// パラメータ化した本体に摂動値を渡す。つまみが効かないメトリクスは現状値のままになり弾力性0になる。
func sensMetrics(master oapi.Raws, s sensScales) []float64 {
	out := make([]float64, 6)

	// 戦闘。敵武器 bite のダメージとプレイヤー筋力を摂動して廃墟 day20 戦力比を測る
	if player, err := LoadCombatantFromMember(master, BaselinePlayer); err == nil {
		if weapon, err := LoadWeaponFromItem(master, BaselineWeapon); err == nil {
			player.Strength = int(math.Round(float64(player.Strength) * s.playerStr))
			withScaledMeleeDamage(master, "bite", s.weapon, func() {
				if curve, e := DifficultyCurve(master, player, weapon, BaselineAreaTable, 20); e == nil && len(curve) >= 20 {
					out[0] = curve[19].PowerRatio
				}
			})
		}
	}
	// 生存。満腹度・減耗ターン・1日ターンを摂動
	out[1] = daysUntilStarvingWith(float64(gc.DefaultMaxHunger)*s.maxHunger, float64(gc.HungerDrainTurns)*s.hungerDrain, float64(gc.TurnsPerDay)*s.turnsPerDay)
	// 疲労。睡眠回復量を摂動
	out[2] = sleepTimeFractionWith(float64(gc.FatigueGainPerTurn), float64(systems.FatigueRecoverPerTurn)*s.fatigueRecover)
	// 物流。燃料熱量を摂動して満載 OIL の航続を測る
	capacity := consts.Milligram(consts.CubeWeightCapacityKg) * consts.MilligramPerKg
	fuel := float64(query.HeatOf(oapi.OIL, capacity)) * s.fuelHeat
	out[3] = DriveRangeTiles(consts.Heat(fuel), capacity)
	// 経済。loot 価値を摂動して1個あたり手取りを測る。送料は固定なので価値に対し弾力性が1を超える
	lv := ExpectedLootValue(master, "ruins_area", 8) * s.lootValue
	if lv > 0 {
		out[4] = float64(query.AuctionNetProceeds(consts.Currency(math.Round(lv)), ExpectedLootWeightKg(master, "ruins_area", 8)))
	}
	// 進行。今回のつまみでは動かさない。他ドメインのつまみが成長に波及しないことを見せる列
	out[5] = float64(AttacksToSkillLevel(0, 30))
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

// CrossDomainSensitivity はドメイン横断の感度行列を返す。各つまみを+10%し、全ドメインの代表
// メトリクスがどれだけ動くかを測る。ヤコビアンで、多くはブロック対角、すなわち各つまみは自分の
// ドメインのメトリクスだけを動かす。横断するのは1日ターン数のような共通の分母だけ。
func CrossDomainSensitivity(master oapi.Raws) []KnobSensitivity {
	knobs := []struct {
		name, domain string
		apply        func(*sensScales)
	}{
		{"敵武器ダメージ(bite)", "戦闘", func(s *sensScales) { s.weapon = 1.1 }},
		{"プレイヤー筋力", "戦闘", func(s *sensScales) { s.playerStr = 1.1 }},
		{"最大満腹度", "生存", func(s *sensScales) { s.maxHunger = 1.1 }},
		{"空腹減耗ターン", "生存", func(s *sensScales) { s.hungerDrain = 1.1 }},
		{"疲労回復量", "疲労", func(s *sensScales) { s.fatigueRecover = 1.1 }},
		{"燃料熱量", "物流", func(s *sensScales) { s.fuelHeat = 1.1 }},
		{"loot価値", "経済", func(s *sensScales) { s.lootValue = 1.1 }},
		{"1日ターン数", "横断", func(s *sensScales) { s.turnsPerDay = 1.1 }},
	}
	base := sensMetrics(master, baseScales())
	rows := make([]KnobSensitivity, 0, len(knobs))
	for _, k := range knobs {
		s := baseScales()
		k.apply(&s)
		pert := sensMetrics(master, s)
		cells := make([]SensitivityCell, len(SensitivityMetricNames))
		for i := range SensitivityMetricNames {
			pct := 0.0
			if base[i] != 0 {
				pct = (pert[i] - base[i]) / base[i] * 100
			}
			cells[i] = SensitivityCell{Metric: SensitivityMetricNames[i], PctChange: pct}
		}
		rows = append(rows, KnobSensitivity{Knob: k.name, Domain: k.domain, Cells: cells})
	}
	return rows
}
