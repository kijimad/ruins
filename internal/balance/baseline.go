package balance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ベースライン導出の基準。プレイヤーは強化なしの素手を最悪ケースとして固定する。cmd と探索で
// 同じ値を参照し、テーブルと実機能がずれるのを防ぐ。
const (
	BaselinePlayer    = "ash"
	BaselineWeapon    = "bare_hands"
	BaselineDays      = 21
	BaselineAreaTable = "ruins_area"
)

// TargetPowerRatio は序盤戦闘の目標帯の中心値を返す。day1=2.5 から day20=1.3 へ線形に下げ、
// 序盤は余裕あり、終盤で拮抗へ寄せる。値は人間が決める設計仮説で、プレイで見直す。
func TargetPowerRatio(day int) float64 {
	const d0, m0, d1, m1 = 1, 2.5, 20, 1.3
	switch {
	case day <= d0:
		return m0
	case day >= d1:
		return m1
	default:
		return m0 + (m1-m0)*float64(day-d0)/float64(d1-d0)
	}
}

// targetBand は目標帯の許容幅。中心±この値を目標帯内とみなす。
const targetBand = 0.3

// BaselineRow はベースライン表の1行。ある日の実測 powerRatio と目標帯の関係を持つ。
type BaselineRow struct {
	Day        int
	Danger     int
	PowerRatio float64
	Target     float64
	InRange    bool
}

// Baseline は指定した敵テーブルの序盤戦闘ベースラインを返す。プレイヤーは強化なしの素手を
// 最悪ケースとして固定し、難易度カーブを目標帯と突き合わせる。
func Baseline(master oapi.Raws, playerName, weaponName, enemyTableName string, days int) ([]BaselineRow, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, err
	}
	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return nil, err
	}
	curve, err := DifficultyCurve(master, player, weapon, enemyTableName, days)
	if err != nil {
		return nil, err
	}
	rows := make([]BaselineRow, len(curve))
	for i, m := range curve {
		tg := TargetPowerRatio(m.Day)
		rows[i] = BaselineRow{
			Day:        m.Day,
			Danger:     m.Danger,
			PowerRatio: m.PowerRatio,
			Target:     tg,
			InRange:    m.PowerRatio >= tg-targetBand && m.PowerRatio <= tg+targetBand,
		}
	}
	return rows, nil
}

// RenderBaselineMarkdown は全敵テーブルのベースラインを markdown 表にする。パラメータ変更の
// 影響が diff で読めるよう、テーブルごとに日次の powerRatio と目標・目標帯内かを並べる。
// 文字列リテラルは源泉英語化の方針に合わせ英語にする。テーブル名は raw 由来の実行時データ。
func RenderBaselineMarkdown(master oapi.Raws, playerName, weaponName string, days int) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# バランスベースライン\n\n")
	fmt.Fprintf(&b, "実コードの戦闘式と raw.toml から乱数なしで導出したバランスの現状値。`make balance-report` で再生成する。\n")
	fmt.Fprintf(&b, "パラメータを変えたときはこのファイルの差分が影響を示す。目標や式の詳細は docs/balance/approach.md を参照。\n\n")

	// 序盤戦闘のカーブ
	fmt.Fprintf(&b, "## 序盤戦闘の難易度カーブ\n\n")
	fmt.Fprintf(&b, "**概要**: 各日にプレイヤーがどれだけ有利かを戦力比で表す。戦力比は敵を倒す速さ÷敵に倒される速さで、1.0が互角、大きいほど楽。\n")
	fmt.Fprintf(&b, "プレイヤーは強化なしの `%s` + `%s` を最悪ケースとして固定する。実プレイは武器強化でこれより楽になる。\n", playerName, weaponName)
	fmt.Fprintf(&b, "目標帯は day1=2.5 から day20=1.3 へ下げ、許容幅±%.1f。目標は設計仮説でプレイで見直す。判定「内」が目標帯の中、「外」が外。\n\n", targetBand)

	tables := raw.PtrSlice(master.EnemyTables)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Id < tables[j].Id })
	for _, table := range tables {
		rows, err := Baseline(master, playerName, weaponName, table.Id, days)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "### %s (%s)\n\n", table.Name, table.Id)
		fmt.Fprintf(&b, "| 日 | 危険度 | 戦力比 | 目標 | 判定 |\n")
		fmt.Fprintf(&b, "|---:|---:|---:|---:|:--:|\n")
		for _, r := range rows {
			mark := "外"
			if r.InRange {
				mark = "内"
			}
			fmt.Fprintf(&b, "| %d | %d | %.2f | %.2f | %s |\n", r.Day, r.Danger, r.PowerRatio, r.Target, mark)
		}
		fmt.Fprintf(&b, "\n")
	}

	// 生存圧
	fmt.Fprintf(&b, "## 生存圧\n\n")
	fmt.Fprintf(&b, "**概要**: 補給なしで生き延びられる時間。食料は日単位、寒さはターン単位で時間スケールが大きく違う。\n")
	fmt.Fprintf(&b, "飢えは満腹度が尽きるまで、寒さは平熱から低体温が発生するまでを表す。実効温度は周囲温度に断熱を足した値。\n\n")
	fmt.Fprintf(&b, "| 指標 | 値 |\n|---|---:|\n")
	fmt.Fprintf(&b, "| 栄養失調まで（満腹度33%%未満）の日数 | %.2f |\n", DaysUntilStarving())
	fmt.Fprintf(&b, "| 満腹度が尽きるまでの日数 | %.2f |\n", DaysUntilHungerEmpty())
	fmt.Fprintf(&b, "| 睡眠なしで疲労までの日数 | %.2f |\n", DaysUntilTired())
	fmt.Fprintf(&b, "| 睡眠なしで過労までの日数 | %.2f |\n", DaysUntilExhausted())
	fmt.Fprintf(&b, "\n| 実効温度(℃) | 低体温までのターン |\n|---:|---:|\n")
	for _, temp := range []int{-20, -10, 0, 5, 10, 15} {
		fmt.Fprintf(&b, "| %d | %.0f |\n", temp, TurnsToHypothermia(temp))
	}
	fmt.Fprintf(&b, "\n")

	// 身体の連鎖。状態異常→血液低下→HP減
	fmt.Fprintf(&b, "## 身体の連鎖（血液→HP）\n\n")
	fmt.Fprintf(&b, "**概要**: 状態異常は血液を下げ、血液が危険域(40)を割ると毎ターンHPが減り始める。連鎖の終端を示す。\n")
	fmt.Fprintf(&b, "上流は状態異常の重症度で、切傷は重症度1段あたり血液-25、低体温は-22。重症(×3)の切傷1つで血液は約25まで落ちる。\n\n")
	fmt.Fprintf(&b, "| 血液 | HP減(/ターン) |\n|---:|---:|\n")
	for _, blood := range []int{100, 50, 40, 30, 20, 10, 0} {
		fmt.Fprintf(&b, "| %d | %d |\n", blood, HPDrainPerTurnAtBlood(blood))
	}
	fmt.Fprintf(&b, "\n")

	// 経済
	fmt.Fprintf(&b, "## 経済（競売の手取り）\n\n")
	fmt.Fprintf(&b, "**概要**: 基準価値どおりに落札されたときの手取り率。手取り=落札額−手数料12%%−発送料25/kg。集荷料は別立て。\n")
	fmt.Fprintf(&b, "重い安物ほど発送料に食われ手取りが下がる。落札額の分散や入札の伸びは確率過程で、実分布はモンテカルロで測る。\n\n")
	fmt.Fprintf(&b, "| 価値 | 重量(kg) | 手取り率 |\n|---:|---:|---:|\n")
	for _, c := range []struct {
		value  consts.Currency
		weight float64
	}{{1000, 0.1}, {1000, 1}, {1000, 3}, {1000, 10}, {200, 3}} {
		fmt.Fprintf(&b, "| %d | %g | %.0f%% |\n", c.value, c.weight, AuctionTakeHomeRate(c.value, c.weight)*100)
	}
	fmt.Fprintf(&b, "\n")

	// 物流
	fmt.Fprintf(&b, "## 物流（燃料・重量・航続）\n\n")
	fmt.Fprintf(&b, "**概要**: キューブがどれだけ走れるか。1タイルの燃費は基準%d+積載1kgごとに%dで、積むほど悪化する。\n", consts.DriveFuelBase, consts.DriveFuelPerKg)
	fmt.Fprintf(&b, "燃料自身も重量になるので積むほど頭打ちになる。容量は%dkg。\n\n", consts.CubeWeightCapacityKg)
	fmt.Fprintf(&b, "| 積載(kg) | 燃費(/タイル) |\n|---:|---:|\n")
	for _, kg := range []int{0, 100, 250, 500} {
		fmt.Fprintf(&b, "| %d | %d |\n", kg, query.DriveFuelCost(consts.Milligram(kg)*consts.MilligramPerKg))
	}
	fmt.Fprintf(&b, "\n| 満載時の燃料 | 航続(タイル) |\n|---|---:|\n")
	fmt.Fprintf(&b, "| OIL 500kg | %.0f |\n", DriveRangeAllFuel(oapi.OIL, consts.CubeWeightCapacityKg))
	fmt.Fprintf(&b, "| WOOD 500kg | %.0f |\n", DriveRangeAllFuel(oapi.WOOD, consts.CubeWeightCapacityKg))
	fmt.Fprintf(&b, "\n積荷とのトレード。OIL燃料250kgのとき、積荷0なら航続%.0f、積荷250kg追加で航続%.0fへ縮む。\n\n",
		DriveRangeTiles(query.HeatOf(oapi.OIL, 250*consts.MilligramPerKg), 250*consts.MilligramPerKg),
		DriveRangeTiles(query.HeatOf(oapi.OIL, 250*consts.MilligramPerKg), 500*consts.MilligramPerKg))

	// 感度
	fmt.Fprintf(&b, "## 感度（廃墟 day20 戦力比、各つまみ+10%%）\n\n")
	fmt.Fprintf(&b, "**概要**: どのパラメータを動かすと難易度が動くかの目安。各武器のダメージを+10%%したとき、廃墟 day20 の戦力比がどれだけ変わるかを示す。\n")
	fmt.Fprintf(&b, "変化が大きいほど効くつまみ。整数ダメージの丸めで小さな値の変化は表に出ないことがある。\n\n")
	fmt.Fprintf(&b, "| つまみ | 現状 | +10%% | 変化 |\n|---|---:|---:|---:|\n")
	for _, s := range SensitivityDay20(master) {
		fmt.Fprintf(&b, "| %s | %.2f | %.2f | %+.1f%% |\n", s.Knob, s.Base, s.Plus10, s.DeltaRatio*100)
	}
	fmt.Fprintf(&b, "\n")
	return b.String(), nil
}
