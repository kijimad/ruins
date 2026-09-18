package balance

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/olekukonko/tablewriter/tw"
)

// ベースライン導出の基準。プレイヤーは強化なしの素手を最悪ケースとして固定する。cmd と探索で
// 同じ値を参照し、テーブルと実機能がずれるのを防ぐ。
const (
	BaselinePlayer    = "ash"
	BaselineWeapon    = "bare_hands"
	BaselineDays      = 21
	BaselineAreaTable = "ruins_area"
)

// colDanger は複数の表で使う危険度列の見出し。文字列の重複を1つにまとめる。
const colDanger = "危険度"

// TargetPowerRatio は素手・無装備の床の戦力比の目標中心値を返す。床は下限リファレンスで主基準ではない。
// 主基準は想定プレイヤーの死亡確率 TargetExpectedDeath。敵を進行で強化する設計では後半に床が下回るのが正常。
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

// TargetExpectedDeath は想定プレイヤーの死亡確率の目標帯を返す。主基準。day1 の [0,2%] から day15 以降の
// [2%,9%] へ線形に開き、序盤は安全・後半に緊張を立てる。値は設計仮説でプレイで見直す。
func TargetExpectedDeath(day int) (lo, hi float64) {
	t := float64(day-1) / 14
	t = max(0, min(1, t))
	return 0.02 * t, 0.02 + 0.07*t
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

// RenderBaselineMarkdown は全敵テーブルのベースラインを markdown 表にする。日次の powerRatio と目標・帯内かを
// 並べ、パラメータ変更の影響を diff で読めるようにする。テーブル名は raw 由来の実行時データ。
func RenderBaselineMarkdown(master oapi.Raws, playerName, weaponName string, days int) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# バランスベースライン\n\n")
	fmt.Fprintf(&b, "実コードの戦闘式と raw.toml から乱数なしで導出したバランスの現状値。`make balance-report` で再生成する。\n")
	fmt.Fprintf(&b, "パラメータを変えたときはこのファイルの差分が影響を示す。目標や式の詳細は docs/balance/approach.md を参照。\n\n")

	// 目標帯ダッシュボード。ドメイン横断でスカラー指標が設計上の許容帯に収まるかを一望する
	fmt.Fprintf(&b, "## 目標帯（ドメイン横断）\n\n")
	fmt.Fprintf(&b, "**概要**: 各ドメインのスカラー指標が設計上の目標帯に収まるか。戦闘は日で変わるので下の難易度カーブ側で内外を持つ。\n")
	fmt.Fprintf(&b, "目標帯は設計仮説でプレイで見直す。静的下限がこの範囲に収まれば実プレイは少なくともこれだけ快適という下限側の目安。\n\n")
	targets := DomainTargets()
	domainRows := make([][]string, 0, len(targets))
	for _, c := range targets {
		mark := "外"
		if c.InRange() {
			mark = "内"
		}
		domainRows = append(domainRows, []string{c.Domain, c.Metric, fmt.Sprintf("%.2f", c.Value), fmt.Sprintf("%.2f〜%.2f", c.Lo, c.Hi), mark})
	}
	writeMDTable(&b, []string{"ドメイン", "指標", "現状", "目標帯", "判定"}, []tw.Align{alignL, alignL, alignR, alignC, alignC}, domainRows)

	// 序盤戦闘のカーブ
	fmt.Fprintf(&b, "## 序盤戦闘の難易度カーブ\n\n")
	fmt.Fprintf(&b, "**概要**: 各日に素手・無装備の床プレイヤーがどれだけ有利かを戦力比で表す。戦力比は敵を倒す速さ÷敵に倒される速さで、1.0が互角、大きいほど楽。\n")
	fmt.Fprintf(&b, "床は難易度の下限リファレンスで主基準ではない。主基準は「進行カーブ」の想定プレイヤーの死亡確率。敵を進行で強化する設計では床は後半に目標帯を下回り 外 になるのが正常で、装備しないと生き残れないことを表す。\n")
	fmt.Fprintf(&b, "プレイヤーは強化なしの `%s` + `%s` に固定する。\n", playerName, weaponName)
	fmt.Fprintf(&b, "目標帯は day1=2.5 から day20=1.3 へ下げ、許容幅±%.1f。目標は設計仮説でプレイで見直す。判定「内」が目標帯の中、「外」が外。\n\n", targetBand)

	tables := raw.PtrSlice(master.EnemyTables)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Id < tables[j].Id })
	for _, table := range tables {
		rows, err := Baseline(master, playerName, weaponName, table.Id, days)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "### %s (%s)\n\n", table.Name, table.Id)
		var curveRows [][]string
		for _, r := range rows {
			mark := "外"
			if r.InRange {
				mark = "内"
			}
			curveRows = append(curveRows, []string{fmt.Sprintf("%d", r.Day), fmt.Sprintf("%d", r.Danger), fmt.Sprintf("%.2f", r.PowerRatio), fmt.Sprintf("%.2f", r.Target), mark})
		}
		writeMDTable(&b, []string{"日", colDanger, "戦力比", "目標", "判定"}, []tw.Align{alignR, alignR, alignR, alignR, alignC}, curveRows)
	}

	// 戦闘リスク。期待値の比でなく、分布から突然死の確率を測る
	if err := renderCombatRisk(&b, master, playerName, weaponName, days); err != nil {
		return "", err
	}

	// 気候。世界温度の季節変動
	fmt.Fprintf(&b, "## 気候（世界温度の季節変動）\n\n")
	fmt.Fprintf(&b, "**概要**: 屋外の世界温度は1年周期で春から夏ピーク、秋、冬底へ巡る。緯度勾配・時間帯・遮蔽を含まない季節そのもの。\n")
	fmt.Fprintf(&b, "冬は準備なしでは生存できない寒さになり、生存圧の寒さ側の上流入力になる。\n\n")
	var climateRows [][]string
	for _, day := range []int{1, 4, 8, 12, 16, 20, 24, 28} {
		climateRows = append(climateRows, []string{fmt.Sprintf("%d", day), fmt.Sprintf("%d", WorldTemperatureAtDay(day))})
	}
	writeMDTable(&b, []string{"経過日", "世界温度(℃)"}, []tw.Align{alignR, alignR}, climateRows)

	// 生存圧
	fmt.Fprintf(&b, "## 生存圧\n\n")
	fmt.Fprintf(&b, "**概要**: 補給なしで生き延びられる時間。食料は日単位、寒さはターン単位で時間スケールが大きく違う。\n")
	fmt.Fprintf(&b, "飢えは満腹度が尽きるまで、寒さは平熱から低体温が発生するまでを表す。実効温度は周囲温度に断熱を足した値。\n\n")
	params := DefaultParams()
	writeMDTable(&b, []string{"指標", "値"}, []tw.Align{alignL, alignR}, [][]string{
		{"栄養失調まで（満腹度33%未満）の日数", fmt.Sprintf("%.2f", DaysUntilStarving(params))},
		{"満腹度が尽きるまでの日数", fmt.Sprintf("%.2f", DaysUntilHungerEmpty(params))},
		{"睡眠なしで疲労までの日数", fmt.Sprintf("%.2f", DaysUntilTired(params))},
		{"睡眠なしで過労までの日数", fmt.Sprintf("%.2f", DaysUntilExhausted(params))},
		{"満タンから睡眠で回復し切るターン(地べた)", fmt.Sprintf("%.0f", SleepTurnsToFullRecover(params))},
		{"釣り合いに要する睡眠時間の割合", fmt.Sprintf("%.0f%%", SleepTimeFraction(params)*100)},
	})
	var hypoRows [][]string
	for _, temp := range []int{-20, -10, 0, 5, 10, 15} {
		hypoRows = append(hypoRows, []string{fmt.Sprintf("%d", temp), fmt.Sprintf("%.0f", TurnsToHypothermia(temp))})
	}
	writeMDTable(&b, []string{"実効温度(℃)", "低体温までのターン"}, []tw.Align{alignR, alignR}, hypoRows)

	// 身体の連鎖。状態異常→血液低下→HP減
	fmt.Fprintf(&b, "## 身体の連鎖（血液→HP）\n\n")
	fmt.Fprintf(&b, "**概要**: 状態異常は血液を下げ、血液が危険域(40)を割ると毎ターンHPが減り始める。連鎖の終端を示す。\n")
	fmt.Fprintf(&b, "上流は状態異常の重症度で、切傷は重症度1段あたり血液-25、低体温は-22。重症(×3)の切傷1つで血液は約25まで落ちる。\n\n")
	var bloodRows [][]string
	for _, blood := range []int{100, 50, 40, 30, 20, 10, 0} {
		bloodRows = append(bloodRows, []string{fmt.Sprintf("%d", blood), fmt.Sprintf("%d", HPDrainPerTurnAtBlood(blood))})
	}
	writeMDTable(&b, []string{"血液", "HP減(/ターン)"}, []tw.Align{alignR, alignR}, bloodRows)

	// 経済
	fmt.Fprintf(&b, "## 経済（競売の手取り）\n\n")
	fmt.Fprintf(&b, "**概要**: 基準価値どおりに落札されたときの手取り率。手取り=落札額−手数料12%%−発送料25/kg。集荷料は別立て。\n")
	fmt.Fprintf(&b, "重い安物ほど発送料に食われ手取りが下がる。落札額の分散や入札の伸びは確率過程で、実分布はモンテカルロで測る。\n\n")
	var takeHomeRows [][]string
	for _, c := range []struct {
		value  consts.Currency
		weight float64
	}{{1000, 0.1}, {1000, 1}, {1000, 3}, {1000, 10}, {200, 3}} {
		takeHomeRows = append(takeHomeRows, []string{fmt.Sprintf("%d", c.value), fmt.Sprintf("%g", c.weight), fmt.Sprintf("%.0f%%", AuctionTakeHomeRate(c.value, c.weight)*100)})
	}
	writeMDTable(&b, []string{"価値", "重量(kg)", "手取り率"}, []tw.Align{alignR, alignR, alignR}, takeHomeRows)

	// 収入側。危険度ごとに拾えるアイテムの期待価値
	fmt.Fprintf(&b, "収入側。危険度ごとに拾える1個あたりの期待価値。危険度が上がるほど高価な loot が出る。\n\n")
	var lootValueRows [][]string
	for _, danger := range []int{1, 3, 5, 8} {
		lootValueRows = append(lootValueRows, []string{fmt.Sprintf("%d", danger),
			fmt.Sprintf("%.0f", ExpectedLootValue(master, "ruins_area", danger)),
			fmt.Sprintf("%.0f", ExpectedLootValue(master, "forest", danger)),
			fmt.Sprintf("%.0f", ExpectedLootValue(master, "cave", danger))})
	}
	writeMDTable(&b, []string{colDanger, "廃墟", "森", "洞窟"}, []tw.Align{alignR, alignR, alignR, alignR}, lootValueRows)

	fmt.Fprintf(&b, "手取り側。上の額面から手数料と発送料を引いた1個あたりの期待手取り。発送料は重量比例なので額面より縮む。\n\n")
	var netLootRows [][]string
	for _, danger := range []int{1, 3, 5, 8} {
		netLootRows = append(netLootRows, []string{fmt.Sprintf("%d", danger),
			fmt.Sprintf("%.0f", ExpectedNetLootValue(master, "ruins_area", danger)),
			fmt.Sprintf("%.0f", ExpectedNetLootValue(master, "forest", danger)),
			fmt.Sprintf("%.0f", ExpectedNetLootValue(master, "cave", danger))})
	}
	writeMDTable(&b, []string{colDanger, "廃墟", "森", "洞窟"}, []tw.Align{alignR, alignR, alignR, alignR}, netLootRows)

	fmt.Fprintf(&b, "探索1回の期待収入。各層で約%d個拾い層の深さを危険度として手取りを積んだ閉形式。層数は生存に依る scenario 入力。\n", expectedItemsPerFloor)
	fmt.Fprintf(&b, "総収支はこの収入から移動燃料コストを引くが、1回の移動タイル数はコードにない設計値なのでコスト側は別途与える。\n\n")
	var runIncomeRows [][]string
	for _, floors := range []int{1, 3, 5, 10} {
		runIncomeRows = append(runIncomeRows, []string{fmt.Sprintf("%d", floors), fmt.Sprintf("%.0f", ExpectedRunLootIncome(master, "ruins_area", floors))})
	}
	writeMDTable(&b, []string{"探索層数", "廃墟の期待収入"}, []tw.Align{alignR, alignR}, runIncomeRows)

	// 経済の進行。支出側の生活費と、日ごとに loot がそれを賄えるか
	fmt.Fprintf(&b, "経済の進行。1日の食費は満腹度減耗を最安の食料で埋め戻す費用で %.0f。日が進むと危険度が上がり loot 手取りも上がるので、\n", CostOfLivingPerDay(master, DefaultParams()))
	fmt.Fprintf(&b, "1日分の食費を賄うのに要る loot 個数が減る。個数が小さいほど、その日の探索は生活費に対して割が良い。\n\n")
	var econProgRows [][]string
	for _, d := range EconomyProgression(master, "ruins_area", days) {
		econProgRows = append(econProgRows, []string{fmt.Sprintf("%d", d.Day), fmt.Sprintf("%d", d.Danger), fmt.Sprintf("%.0f", d.NetLootValue), fmt.Sprintf("%.2f", d.LootPerDayFood)})
	}
	writeMDTable(&b, []string{"日", colDanger, "1個あたり手取り", "1日分の食費に要る loot 個数"}, []tw.Align{alignR, alignR, alignR, alignR}, econProgRows)

	// 物流
	fmt.Fprintf(&b, "## 物流（燃料・重量・航続）\n\n")
	fmt.Fprintf(&b, "**概要**: キューブがどれだけ走れるか。1タイルの燃費は基準%d+積載1kgごとに%dで、積むほど悪化する。\n", consts.DriveFuelBase, consts.DriveFuelPerKg)
	fmt.Fprintf(&b, "燃料自身も重量になるので積むほど頭打ちになる。容量は%dkg。\n\n", consts.CubeWeightCapacityKg)
	var fuelCostRows [][]string
	for _, kg := range []int{0, 100, 250, 500} {
		fuelCostRows = append(fuelCostRows, []string{fmt.Sprintf("%d", kg), fmt.Sprintf("%d", query.DriveFuelCost(consts.Milligram(kg)*consts.MilligramPerKg))})
	}
	writeMDTable(&b, []string{"積載(kg)", "燃費(/タイル)"}, []tw.Align{alignR, alignR}, fuelCostRows)
	writeMDTable(&b, []string{"満載時の燃料", "航続(タイル)"}, []tw.Align{alignL, alignR}, [][]string{
		{"OIL 500kg", fmt.Sprintf("%.0f", DriveRangeAllFuel(oapi.OIL, consts.CubeWeightCapacityKg))},
		{"WOOD 500kg", fmt.Sprintf("%.0f", DriveRangeAllFuel(oapi.WOOD, consts.CubeWeightCapacityKg))},
	})
	fmt.Fprintf(&b, "積荷とのトレード。OIL燃料250kgのとき、積荷0なら航続%.0f、積荷250kg追加で航続%.0fへ縮む。\n\n",
		DriveRangeTiles(query.HeatOf(oapi.OIL, 250*consts.MilligramPerKg), 250*consts.MilligramPerKg),
		DriveRangeTiles(query.HeatOf(oapi.OIL, 250*consts.MilligramPerKg), 500*consts.MilligramPerKg))

	fmt.Fprintf(&b, "同じ燃料は火にくべると燃焼ターンになる。地面直の火は熱量の半分を時間へ変える。\n\n")
	writeMDTable(&b, []string{"燃料", "燃焼ターン"}, []tw.Align{alignL, alignR}, [][]string{
		{"OIL 10kg", fmt.Sprintf("%d", FuelBurnTurns(oapi.OIL, 10))},
		{"WOOD 10kg", fmt.Sprintf("%d", FuelBurnTurns(oapi.WOOD, 10))},
	})

	// 感度行列。つまみ×メトリクスの横断ヤコビアン
	renderSensitivityMatrix(&b, master)

	// 交換レート。同じメトリクスを保つためのつまみ間の相互補償
	renderExchangeRates(&b, master)

	// 境界マージン。目標帯を割るまでのつまみの余白
	renderBoundaryMargins(&b, master)

	// 武器・スキルの各分析。劣化量・viability・スキル深度をまとめて書き出す
	if err := renderWeaponSkillSections(&b, master); err != nil {
		return "", err
	}

	// ビルド別のブレ。静的下限に加え、スキル・装備・バフで戦力が振れる幅を見る
	if builds, err := RepresentativeBuilds(master); err == nil {
		fmt.Fprintf(&b, "## ビルド別のブレ（廃墟）\n\n")
		fmt.Fprintf(&b, "**概要**: 強化なしの下限に加え、想定攻撃回数から導いたスキル熟練度でダメージが伸びたビルドを重ね、戦力比が振れる幅を見る。\n")
		names := make([]string, 0, len(builds))
		for _, bp := range builds {
			names = append(names, bp.Name)
		}
		fmt.Fprintf(&b, "ビルドは %v。倍率は恣意的でなくスキル成長と熟練度から導く。幅が広いほどビルド次第で難易度が振れ、終盤で許容を超えるならバフが強すぎる。\n\n", names)
		if spread, err := BuildSpread(master, builds, "ruins_area", 21); err == nil {
			var spreadRows [][]string
			for _, r := range spread {
				if r.Day%4 == 1 || r.Day == 21 {
					spreadRows = append(spreadRows, []string{fmt.Sprintf("%d", r.Day), fmt.Sprintf("%.2f", r.MinRatio), fmt.Sprintf("%.2f", r.MaxRatio), fmt.Sprintf("%.2f", r.Spread())})
				}
			}
			writeMDTable(&b, []string{"日", "下限の戦力比", "最強の戦力比", "幅"}, []tw.Align{alignR, alignR, alignR, alignR}, spreadRows)
		}
		if stages, err := StageSpreads(master, builds, "ruins_area"); err == nil {
			fmt.Fprintf(&b, "段階ごとのブレ幅と許容帯。終盤ほど許容を絞る。判定「外」はビルド次第で振れすぎる段階。許容帯は設計仮説。\n\n")
			var stageRows [][]string
			for _, s := range stages {
				mark := "外"
				if s.InRange() {
					mark = "内"
				}
				stageRows = append(stageRows, []string{s.Stage, fmt.Sprintf("%d-%d", s.FromDay, s.ToDay), fmt.Sprintf("%.2f", s.MaxSpread), fmt.Sprintf("%.2f", s.Tolerance), mark})
			}
			writeMDTable(&b, []string{"段階", "日", "最大の幅", "許容", "判定"}, []tw.Align{alignL, alignL, alignR, alignR, alignC}, stageRows)
		}
	}

	// 多目的探索。序盤・中盤・終盤の目標逸脱を同時に小さくする武器倍率の非劣集合
	fmt.Fprintf(&b, "## 多目的探索（Pareto 前線）\n\n")
	fmt.Fprintf(&b, "**概要**: プレイヤー武器 bare_hands と敵武器 bite のダメージ倍率を格子で動かし、day1・day10・day20 の目標戦力比からの逸脱を同時に見る。\n")
	fmt.Fprintf(&b, "単変数探索は day20 しか合わせられないが、複数日はトレードオフになる。どれかを詰めると別が緩む非劣な組だけを載せる。逸脱は小さいほど目標に近い。\n\n")
	paretoKnobs := []string{BaselineWeapon, "bite"}
	front := ParetoFront(master, paretoKnobs, []float64{0.8, 1.0, 1.5}, []int{1, 10, 20})
	sort.Slice(front, func(i, j int) bool {
		if front[i].Factors[paretoKnobs[0]] != front[j].Factors[paretoKnobs[0]] {
			return front[i].Factors[paretoKnobs[0]] < front[j].Factors[paretoKnobs[0]]
		}
		return front[i].Factors[paretoKnobs[1]] < front[j].Factors[paretoKnobs[1]]
	})
	var paretoRows [][]string
	for _, p := range front {
		paretoRows = append(paretoRows, []string{
			fmt.Sprintf("%.1f", p.Factors[paretoKnobs[0]]), fmt.Sprintf("%.1f", p.Factors[paretoKnobs[1]]),
			fmt.Sprintf("%.2f", p.Deviations[0]), fmt.Sprintf("%.2f", p.Deviations[1]), fmt.Sprintf("%.2f", p.Deviations[2]),
		})
	}
	writeMDTable(&b, []string{"bare_hands倍率", "bite倍率", "day1逸脱", "day10逸脱", "day20逸脱"}, []tw.Align{alignR, alignR, alignR, alignR, alignR}, paretoRows)

	// 進行・成長。スキルを上げるのに要する攻撃回数
	fmt.Fprintf(&b, "## 進行・成長（スキル）\n\n")
	fmt.Fprintf(&b, "**概要**: 武器スキルを上げるのに要する攻撃回数。1攻撃ごとに減衰しながら経験値が入り、スキル値が高いほど遅くなる。\n")
	fmt.Fprintf(&b, "能力値が高いほど成長が速い。ここは静的下限として攻撃回数だけを見る。経過日への写像は攻撃頻度に依るので載せない。\n\n")
	var skillRows [][]string
	for _, abil := range []int{0, 5, 10} {
		skillRows = append(skillRows, []string{fmt.Sprintf("%d", abil),
			fmt.Sprintf("%d", AttacksToSkillLevel(abil, 10)), fmt.Sprintf("%d", AttacksToSkillLevel(abil, 30)), fmt.Sprintf("%d", AttacksToSkillLevel(abil, 50)),
			fmt.Sprintf("%d", SkillLevelAfterAttacks(abil, 100))})
	}
	writeMDTable(&b, []string{"能力値", "Lv10まで", "Lv30まで", "Lv50まで", "100攻撃で到達Lv"}, []tw.Align{alignR, alignR, alignR, alignR, alignR}, skillRows)
	return b.String(), nil
}

// renderCombatRisk は戦闘リスクのカーブを markdown で書き出す。難易度カーブが期待値の比を見るのに対し、
// こちらは吸収マルコフ連鎖で解いた死亡確率と決着ターンを日ごとに出す。突然死の裾を可視化する。
func renderCombatRisk(b *strings.Builder, master oapi.Raws, playerName, weaponName string, days int) error {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return err
	}
	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return err
	}
	curve, err := CombatRiskCurve(master, player, weapon, BaselineAreaTable, days)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "## 戦闘リスク（死亡確率・廃墟）\n\n")
	fmt.Fprintf(b, "**概要**: その日の敵プールに1体遭遇したときに倒される確率と、決着までの期待ターン。戦力比が期待値の比なのに対し、\n")
	fmt.Fprintf(b, "こちらは戦闘を吸収マルコフ連鎖として厳密に解き、平均では見えない突然死の裾を測る。プレイヤーは強化なしの `%s` + `%s` 固定。\n\n", playerName, weaponName)
	var riskRows [][]string
	for _, d := range curve {
		riskRows = append(riskRows, []string{fmt.Sprintf("%d", d.Day), fmt.Sprintf("%d", d.Danger), fmt.Sprintf("%.1f%%", d.DeathProb*100), fmt.Sprintf("%.1f", d.ExpTurns)})
	}
	writeMDTable(b, []string{"日", colDanger, "死亡確率", "期待決着ターン"}, []tw.Align{alignR, alignR, alignR, alignR}, riskRows)
	return renderProgression(b, master, player, weapon, days)
}

// renderProgression は静的下限と想定プレイヤーの進行カーブを markdown で書き出す。難易度の側は日→危険度で
// 進み、プレイヤーの側は想定攻撃頻度からその日の想定スキルで進む。床と想定の帯で進行度調整を見る。
func renderProgression(b *strings.Builder, master oapi.Raws, player CombatantStats, weapon WeaponStats, days int) error {
	curve, err := ProgressionCurve(master, player, weapon, BaselineAreaTable, days, DefaultAttacksPerDay)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "## 進行カーブ（床 vs 想定プレイヤー・廃墟）\n\n")
	fmt.Fprintf(b, "**概要**: 難易度の側は日→危険度で進み、プレイヤーの側は1日%d攻撃の仮説からその日の想定スキルで進む。\n", DefaultAttacksPerDay)
	fmt.Fprintf(b, "床はスキル0・無装備の最悪ケース、想定はその日までに育ったスキルと装備防御を織り込んだ体験。主基準は想定の死亡確率で、目標帯 TargetExpectedDeath と照合して 内/外 を出す。床は難易度の下限リファレンスで、後半は目標を下回るハードモードになるのが正常。攻撃頻度と装備防御は設計仮説。\n\n")
	var rows [][]string
	for _, d := range curve {
		lo, hi := TargetExpectedDeath(d.Day)
		mark := "外"
		if d.DeathExpected >= lo && d.DeathExpected <= hi {
			mark = "内"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", d.Day), fmt.Sprintf("%d", d.Danger), fmt.Sprintf("%d", d.SkillLevel), fmt.Sprintf("+%d", d.GearDefense),
			fmt.Sprintf("%.1f%%", d.DeathFloor*100), fmt.Sprintf("%.1f%%", d.DeathExpected*100), fmt.Sprintf("%.0f〜%.0f%%", lo*100, hi*100),
			mark, fmt.Sprintf("%.1f", d.TurnsFloor), fmt.Sprintf("%.1f", d.TurnsExpected),
		})
	}
	writeMDTable(b, []string{"日", colDanger, "想定Lv", "想定防御", "死亡(床)", "死亡(想定)", "想定の目標帯", "判定", "決着ターン(床)", "決着ターン(想定)"}, []tw.Align{alignR, alignR, alignR, alignR, alignR, alignR, alignC, alignC, alignR, alignR}, rows)
	fmt.Fprintf(b, "想定の死亡確率が目標帯に収まれば、成長込みでも後半に狙った緊張が保たれている。床は無装備・無成長の下限で、後半に目標を下回る=装備しないと生き残れないことを表す。\n\n")
	return nil
}

// renderSensitivityMatrix はドメイン横断の感度行列を markdown で書き出す。RenderBaselineMarkdown の
// 複雑度を抑えるため別関数にする。
func renderSensitivityMatrix(b *strings.Builder, master oapi.Raws) {
	fmt.Fprintf(b, "## 感度行列（各つまみ+10%%、ドメイン横断）\n\n")
	fmt.Fprintf(b, "**概要**: どのつまみがどのメトリクスをどれだけ動かすか。各つまみを+10%%したときの各メトリクスの変化率。\n")
	fmt.Fprintf(b, "多くはブロック対角、すなわち各つまみは自分のドメインだけを動かす。横断するのは1日ターン数のような共通の分母だけ。整数丸めで小さな値は表に出ないことがある。\n\n")
	header := append([]string{"つまみ(ドメイン)"}, SensitivityMetricNames...)
	aligns := make([]tw.Align, 1+len(SensitivityMetricNames))
	aligns[0] = alignL
	for i := 1; i < len(aligns); i++ {
		aligns[i] = alignR
	}
	knobs := CrossDomainSensitivity(master)
	rows := make([][]string, 0, len(knobs))
	for _, k := range knobs {
		row := []string{fmt.Sprintf("%s(%s)", k.Knob, k.Domain)}
		for _, c := range k.Cells {
			if c.PctChange == 0 {
				row = append(row, "-")
			} else {
				row = append(row, fmt.Sprintf("%+.1f%%", c.PctChange))
			}
		}
		rows = append(rows, row)
	}
	writeMDTable(b, header, aligns, rows)
}

// renderExchangeRates はメトリクスごとの交換レート表を markdown で書き出す。行のつまみを+10%した
// とき、列のつまみをどれだけ動かせば同じメトリクスに据え置けるかの相互補償を示す。
func renderExchangeRates(b *strings.Builder, master oapi.Raws) {
	fmt.Fprintf(b, "## 交換レート（相互補償）\n\n")
	fmt.Fprintf(b, "**概要**: あるつまみを+10%%したとき、同じメトリクスを元へ戻すには別のつまみをどれだけ動かせばよいか。\n")
	fmt.Fprintf(b, "行のつまみを+10%%し、列のつまみの変化率で打ち消す。実式を二分探索で解いた値で、感度の比の線形近似より正確。\n")
	fmt.Fprintf(b, "そのメトリクスを動かすつまみが2つ以上あるときだけ表を出す。負は相手を減らして打ち消すことを表す。範囲外で両立不能なら「×」。\n\n")
	for _, mx := range ExchangeRates(master) {
		fmt.Fprintf(b, "### %s\n\n", mx.Metric)
		header := append([]string{"+10%↓ \\ 補償→"}, mx.Knobs...)
		aligns := make([]tw.Align, 1+len(mx.Knobs))
		aligns[0] = alignL
		for i := 1; i < len(aligns); i++ {
			aligns[i] = alignR
		}
		rows := make([][]string, 0, len(mx.Rows))
		for _, row := range mx.Rows {
			cells := []string{row.Knob}
			for _, c := range row.Cells {
				switch {
				case c.Compensator == row.Knob:
					cells = append(cells, "-")
				case !c.OK:
					cells = append(cells, "×")
				default:
					cells = append(cells, fmt.Sprintf("%+.1f%%", c.PctChange))
				}
			}
			rows = append(rows, cells)
		}
		writeMDTable(b, header, aligns, rows)
	}
}

// renderBoundaryMargins は目標帯の余白を markdown で書き出す。各つまみを動かして目標帯を割るまでの
// 変化率を、崖の近い順に並べる。凍結ゲートの点固定を余白監視へ拡張したもの。
func renderBoundaryMargins(b *strings.Builder, master oapi.Raws) {
	margins := BoundaryMargins(master)
	sort.Slice(margins, func(i, j int) bool {
		if margins[i].OK != margins[j].OK {
			return margins[i].OK
		}
		return math.Abs(margins[i].NearestPct) < math.Abs(margins[j].NearestPct)
	})
	fmt.Fprintf(b, "## 目標帯の余白（境界マージン）\n\n")
	fmt.Fprintf(b, "**概要**: 各つまみを動かしたとき、メトリクスが目標帯を割るまでの余白。崖の近い順に並ぶ。\n")
	fmt.Fprintf(b, "凍結ゲートが現在値の点を固定するのに対し、こちらは崖までの距離を測る。余白が小さいつまみほど、少しの調整で帯を外れる。範囲内で端に届かなければ「遠い」。\n\n")
	rows := make([][]string, 0, len(margins))
	for _, m := range margins {
		if !m.OK {
			rows = append(rows, []string{m.Metric, m.Knob, "-", "遠い"})
			continue
		}
		rows = append(rows, []string{m.Metric, m.Knob, m.Edge, fmt.Sprintf("%+.1f%%", m.NearestPct)})
	}
	writeMDTable(b, []string{"メトリクス", "つまみ", "最寄りの端", "余白"}, []tw.Align{alignL, alignL, alignC, alignR}, rows)
}

// renderWeaponRestriction は武器を素手へ制限したときの死亡確率の劣化量を書き出す。劣化が大きい順に必須な武器を、
// 小さい順に効かない死にコンテンツ候補を見せる。Restricted Play の翻案。
func renderWeaponRestriction(b *strings.Builder, master oapi.Raws) error {
	const day = 20
	values, baseDeath, err := WeaponRestrictionValues(master, BaselineAreaTable, day)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	fmt.Fprintf(b, "## 要素の劣化量（武器を素手へ制限・廃墟day%d）\n\n", day)
	fmt.Fprintf(b, "**概要**: 各武器(近接・遠距離)を持たせたときの死亡確率と、素手へ制限したときの死亡確率の上昇量。上昇量が大きいほどその武器は必須で、\n")
	fmt.Fprintf(b, "ゼロに近いほど素手と大差ない死にコンテンツ候補。素手の死亡確率は %.1f%%。Restricted Play をサバイバルへ翻案した指標。\n", baseDeath*100)
	fmt.Fprintf(b, "遠距離武器は弾薬の消費と費用を含めないので、その劣化量は弾薬コストを無視した上限として読む。\n\n")

	const top, bottom = 15, 5
	var rows [][]string
	for i, v := range values {
		if len(values) > top+bottom && i >= top && i < len(values)-bottom {
			continue
		}
		if i == top && len(values) > top+bottom {
			rows = append(rows, []string{fmt.Sprintf("… (%d件省略)", len(values)-top-bottom), "", "", ""})
		}
		rows = append(rows, []string{v.Element, fmt.Sprintf("%.1f%%", v.DeathProbWith*100), fmt.Sprintf("%+.1f%%", v.Degradation*100), fmt.Sprintf("%.1f", v.ExpTurnsWith)})
	}
	writeMDTable(b, []string{"武器", "死亡確率", "劣化量(素手比)", "期待決着ターン"}, []tw.Align{alignL, alignR, alignR, alignR}, rows)
	if len(values) > top+bottom {
		fmt.Fprintf(b, "全%d武器のうち上位%d件と下位%d件を表示。\n\n", len(values), top, bottom)
	} else {
		fmt.Fprintf(b, "全%d武器を表示。\n\n", len(values))
	}
	return nil
}

// renderWeaponSkillSections は武器・スキルの各分析を順に書き出す。RenderBaselineMarkdown の複雑度を
// 抑えるため、エラーを返すレンダラをまとめる。
func renderWeaponSkillSections(b *strings.Builder, master oapi.Raws) error {
	if err := renderWeaponRestriction(b, master); err != nil {
		return err
	}
	if err := renderViability(b, master); err != nil {
		return err
	}
	return renderSkillDepth(b, master)
}

// renderViability は武器ロスターの viability を markdown で書き出す。Pfau の知見に基づき、使える武器が
// 多く、罠が少なく、viable どうしに差がある状態を健全とみなす。全ビルド等価な symmetry は避ける。
func renderViability(b *strings.Builder, master oapi.Raws) error {
	const day = 20
	s, err := WeaponViability(master, BaselineAreaTable, day)
	if err != nil {
		return err
	}
	if s.Total == 0 {
		return nil
	}
	fmt.Fprintf(b, "## 武器の viability（廃墟day%d）\n\n", day)
	fmt.Fprintf(b, "**概要**: プレイヤーは全選択肢が等価な symmetry を嫌い、どれも使えるが差がある viability を好む。\n")
	fmt.Fprintf(b, "死亡確率が %.0f%%以下を viable とし、素手より弱いものを罠として数える。viable どうしの決着ターンに幅があれば選択に意味がある。\n\n", s.Ceiling*100)
	verdict := "symmetry寄り。選択の差が小さい"
	if s.Distinct() {
		verdict = "viabilityあり。使える武器に差がある"
	}
	writeMDTable(b, []string{"指標", "値"}, []tw.Align{alignL, alignR}, [][]string{
		{"評価した武器(近接・遠距離)", fmt.Sprintf("%d", s.Total)},
		{fmt.Sprintf("viable（死亡確率%.0f%%以下）", s.Ceiling*100), fmt.Sprintf("%d (%.0f%%)", s.Viable, s.ViableRate()*100)},
		{"罠（素手より弱い）", fmt.Sprintf("%d", s.Traps)},
		{"viable の決着ターン幅", fmt.Sprintf("%.1f〜%.1f", s.ViableTTKMin, s.ViableTTKMax)},
		{"判定", verdict},
	})
	fmt.Fprintf(b, "罠は装備すると素手より不利になる死にコンテンツ候補。viable が多く差があるほど、ビルドの選択が意味を持つ。\n\n")
	return nil
}

// renderSkillDepth はスキル進行の手応えを markdown で書き出す。弱い武器と強い武器で、レベルアップが
// 体験に響く実効ティアと、丸めで死んだティアの割合を対比する。
func renderSkillDepth(b *strings.Builder, master oapi.Raws) error {
	const day = 20
	fmt.Fprintf(b, "## スキル深度（レベルアップの手応え・廃墟day%d）\n\n", day)
	fmt.Fprintf(b, "**概要**: 素手スキルを1レベル上げるたびに、敵プールの撃破ターンがどれだけ縮むか。熟練度倍率は実ゲームと同じく\n")
	fmt.Fprintf(b, "能力+ダイス+武器の base 全体へ切り捨てで掛かる。撃破ターンが動かないレベルは体験に響かない死んだティア。NTBEA のティア識別性を翻案した指標。\n\n")

	var rows [][]string
	weapons := []string{BaselineWeapon, "iron_sword"}
	for _, wn := range weapons {
		prof, err := SkillDepthProfileFor(master, wn, BaselineAreaTable, day)
		if err != nil {
			return err
		}
		total := len(prof.Tiers) - 1
		if total < 1 {
			continue
		}
		gap := "-"
		if prof.EffectiveSteps > 0 {
			gap = fmt.Sprintf("%.3f", prof.MinEffectiveGap)
		}
		rows = append(rows, []string{wn, fmt.Sprintf("%d", total), fmt.Sprintf("%d", prof.EffectiveSteps), fmt.Sprintf("%d", prof.DeadTiers), gap})
	}
	writeMDTable(b, []string{"武器", "総レベル", "実効ティア", "死んだティア", "実効の最小改善(ターン)"}, []tw.Align{alignL, alignR, alignR, alignR, alignR}, rows)
	fmt.Fprintf(b, "熟練度が base 全体へ効くので進行の大半は手応えがある。死んだティアは撃破ターンが既に短い高レベル帯に偏る。弱い武器ほど base が小さく、切り捨てでわずかに死にやすい。\n\n")
	return nil
}
