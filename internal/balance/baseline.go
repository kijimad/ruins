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

	// 目標帯ダッシュボード。ドメイン横断でスカラー指標が設計上の許容帯に収まるかを一望する
	fmt.Fprintf(&b, "## 目標帯（ドメイン横断）\n\n")
	fmt.Fprintf(&b, "**概要**: 各ドメインのスカラー指標が設計上の目標帯に収まるか。戦闘は日で変わるので下の難易度カーブ側で内外を持つ。\n")
	fmt.Fprintf(&b, "目標帯は設計仮説でプレイで見直す。静的下限がこの範囲に収まれば実プレイは少なくともこれだけ快適という下限側の目安。\n\n")
	fmt.Fprintf(&b, "| ドメイン | 指標 | 現状 | 目標帯 | 判定 |\n|---|---|---:|:--:|:--:|\n")
	for _, c := range DomainTargets() {
		mark := "外"
		if c.InRange() {
			mark = "内"
		}
		fmt.Fprintf(&b, "| %s | %s | %.2f | %.2f〜%.2f | %s |\n", c.Domain, c.Metric, c.Value, c.Lo, c.Hi, mark)
	}
	fmt.Fprintf(&b, "\n")

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

	// 戦闘リスク。期待値の比でなく、分布から突然死の確率を測る
	if err := renderCombatRisk(&b, master, playerName, weaponName, days); err != nil {
		return "", err
	}

	// 気候。世界温度の季節変動
	fmt.Fprintf(&b, "## 気候（世界温度の季節変動）\n\n")
	fmt.Fprintf(&b, "**概要**: 屋外の世界温度は1年周期で春から夏ピーク、秋、冬底へ巡る。緯度勾配・時間帯・遮蔽を含まない季節そのもの。\n")
	fmt.Fprintf(&b, "冬は準備なしでは生存できない寒さになり、生存圧の寒さ側の上流入力になる。\n\n")
	fmt.Fprintf(&b, "| 経過日 | 世界温度(℃) |\n|---:|---:|\n")
	for _, day := range []int{1, 4, 8, 12, 16, 20, 24, 28} {
		fmt.Fprintf(&b, "| %d | %d |\n", day, WorldTemperatureAtDay(day))
	}
	fmt.Fprintf(&b, "\n")

	// 生存圧
	fmt.Fprintf(&b, "## 生存圧\n\n")
	fmt.Fprintf(&b, "**概要**: 補給なしで生き延びられる時間。食料は日単位、寒さはターン単位で時間スケールが大きく違う。\n")
	fmt.Fprintf(&b, "飢えは満腹度が尽きるまで、寒さは平熱から低体温が発生するまでを表す。実効温度は周囲温度に断熱を足した値。\n\n")
	fmt.Fprintf(&b, "| 指標 | 値 |\n|---|---:|\n")
	params := DefaultParams()
	fmt.Fprintf(&b, "| 栄養失調まで（満腹度33%%未満）の日数 | %.2f |\n", DaysUntilStarving(params))
	fmt.Fprintf(&b, "| 満腹度が尽きるまでの日数 | %.2f |\n", DaysUntilHungerEmpty(params))
	fmt.Fprintf(&b, "| 睡眠なしで疲労までの日数 | %.2f |\n", DaysUntilTired(params))
	fmt.Fprintf(&b, "| 睡眠なしで過労までの日数 | %.2f |\n", DaysUntilExhausted(params))
	fmt.Fprintf(&b, "| 満タンから睡眠で回復し切るターン(地べた) | %.0f |\n", SleepTurnsToFullRecover(params))
	fmt.Fprintf(&b, "| 釣り合いに要する睡眠時間の割合 | %.0f%% |\n", SleepTimeFraction(params)*100)
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

	// 収入側。危険度ごとに拾えるアイテムの期待価値
	fmt.Fprintf(&b, "\n収入側。危険度ごとに拾える1個あたりの期待価値。危険度が上がるほど高価な loot が出る。\n\n")
	fmt.Fprintf(&b, "| 危険度 | %s | %s | %s |\n|---:|---:|---:|---:|\n", "廃墟", "森", "洞窟")
	for _, danger := range []int{1, 3, 5, 8} {
		fmt.Fprintf(&b, "| %d | %.0f | %.0f | %.0f |\n", danger,
			ExpectedLootValue(master, "ruins_area", danger),
			ExpectedLootValue(master, "forest", danger),
			ExpectedLootValue(master, "cave", danger))
	}

	fmt.Fprintf(&b, "\n手取り側。上の額面から手数料と発送料を引いた1個あたりの期待手取り。発送料は重量比例なので額面より縮む。\n\n")
	fmt.Fprintf(&b, "| 危険度 | 廃墟 | 森 | 洞窟 |\n|---:|---:|---:|---:|\n")
	for _, danger := range []int{1, 3, 5, 8} {
		fmt.Fprintf(&b, "| %d | %.0f | %.0f | %.0f |\n", danger,
			ExpectedNetLootValue(master, "ruins_area", danger),
			ExpectedNetLootValue(master, "forest", danger),
			ExpectedNetLootValue(master, "cave", danger))
	}

	fmt.Fprintf(&b, "\n探索1回の期待収入。各層で約%d個拾い層の深さを危険度として手取りを積んだ閉形式。層数は生存に依る scenario 入力。\n", expectedItemsPerFloor)
	fmt.Fprintf(&b, "総収支はこの収入から移動燃料コストを引くが、1回の移動タイル数はコードにない設計値なのでコスト側は別途与える。\n\n")
	fmt.Fprintf(&b, "| 探索層数 | 廃墟の期待収入 |\n|---:|---:|\n")
	for _, floors := range []int{1, 3, 5, 10} {
		fmt.Fprintf(&b, "| %d | %.0f |\n", floors, ExpectedRunLootIncome(master, "ruins_area", floors))
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

	fmt.Fprintf(&b, "同じ燃料は火にくべると燃焼ターンになる。地面直の火は熱量の半分を時間へ変える。\n\n")
	fmt.Fprintf(&b, "| 燃料 | 燃焼ターン |\n|---|---:|\n")
	fmt.Fprintf(&b, "| OIL 10kg | %d |\n", FuelBurnTurns(oapi.OIL, 10))
	fmt.Fprintf(&b, "| WOOD 10kg | %d |\n", FuelBurnTurns(oapi.WOOD, 10))
	fmt.Fprintf(&b, "\n")

	// 感度行列。つまみ×メトリクスの横断ヤコビアン
	renderSensitivityMatrix(&b, master)

	// 交換レート。同じメトリクスを保つためのつまみ間の相互補償
	renderExchangeRates(&b, master)

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
			fmt.Fprintf(&b, "| 日 | 下限の戦力比 | 最強の戦力比 | 幅 |\n|---:|---:|---:|---:|\n")
			for _, r := range spread {
				if r.Day%4 == 1 || r.Day == 21 {
					fmt.Fprintf(&b, "| %d | %.2f | %.2f | %.2f |\n", r.Day, r.MinRatio, r.MaxRatio, r.Spread())
				}
			}
			fmt.Fprintf(&b, "\n")
		}
		if stages, err := StageSpreads(master, builds, "ruins_area"); err == nil {
			fmt.Fprintf(&b, "段階ごとのブレ幅と許容帯。終盤ほど許容を絞る。判定「外」はビルド次第で振れすぎる段階。許容帯は設計仮説。\n\n")
			fmt.Fprintf(&b, "| 段階 | 日 | 最大の幅 | 許容 | 判定 |\n|---|---|---:|---:|:--:|\n")
			for _, s := range stages {
				mark := "外"
				if s.InRange() {
					mark = "内"
				}
				fmt.Fprintf(&b, "| %s | %d-%d | %.2f | %.2f | %s |\n", s.Stage, s.FromDay, s.ToDay, s.MaxSpread, s.Tolerance, mark)
			}
			fmt.Fprintf(&b, "\n")
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
	fmt.Fprintf(&b, "| bare_hands倍率 | bite倍率 | day1逸脱 | day10逸脱 | day20逸脱 |\n|---:|---:|---:|---:|---:|\n")
	for _, p := range front {
		fmt.Fprintf(&b, "| %.1f | %.1f | %.2f | %.2f | %.2f |\n",
			p.Factors[paretoKnobs[0]], p.Factors[paretoKnobs[1]], p.Deviations[0], p.Deviations[1], p.Deviations[2])
	}
	fmt.Fprintf(&b, "\n")

	// 進行・成長。スキルを上げるのに要する攻撃回数
	fmt.Fprintf(&b, "## 進行・成長（スキル）\n\n")
	fmt.Fprintf(&b, "**概要**: 武器スキルを上げるのに要する攻撃回数。1攻撃ごとに減衰しながら経験値が入り、スキル値が高いほど遅くなる。\n")
	fmt.Fprintf(&b, "能力値が高いほど成長が速い。ここは静的下限として攻撃回数だけを見る。経過日への写像は攻撃頻度に依るので載せない。\n\n")
	fmt.Fprintf(&b, "| 能力値 | Lv10まで | Lv30まで | Lv50まで | 100攻撃で到達Lv |\n|---:|---:|---:|---:|---:|\n")
	for _, abil := range []int{0, 5, 10} {
		fmt.Fprintf(&b, "| %d | %d | %d | %d | %d |\n", abil,
			AttacksToSkillLevel(abil, 10), AttacksToSkillLevel(abil, 30), AttacksToSkillLevel(abil, 50),
			SkillLevelAfterAttacks(abil, 100))
	}
	fmt.Fprintf(&b, "\n")
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
	fmt.Fprintf(b, "| 日 | 危険度 | 死亡確率 | 期待決着ターン |\n|---:|---:|---:|---:|\n")
	for _, d := range curve {
		fmt.Fprintf(b, "| %d | %d | %.1f%% | %.1f |\n", d.Day, d.Danger, d.DeathProb*100, d.ExpTurns)
	}
	fmt.Fprintf(b, "\n")
	return nil
}

// renderSensitivityMatrix はドメイン横断の感度行列を markdown で書き出す。RenderBaselineMarkdown の
// 複雑度を抑えるため別関数にする。
func renderSensitivityMatrix(b *strings.Builder, master oapi.Raws) {
	fmt.Fprintf(b, "## 感度行列（各つまみ+10%%、ドメイン横断）\n\n")
	fmt.Fprintf(b, "**概要**: どのつまみがどのメトリクスをどれだけ動かすか。各つまみを+10%%したときの各メトリクスの変化率。\n")
	fmt.Fprintf(b, "多くはブロック対角、すなわち各つまみは自分のドメインだけを動かす。横断するのは1日ターン数のような共通の分母だけ。整数丸めで小さな値は表に出ないことがある。\n\n")
	fmt.Fprintf(b, "| つまみ(ドメイン)")
	for _, m := range SensitivityMetricNames {
		fmt.Fprintf(b, " | %s", m)
	}
	fmt.Fprintf(b, " |\n|---")
	for range SensitivityMetricNames {
		fmt.Fprintf(b, "|---:")
	}
	fmt.Fprintf(b, "|\n")
	for _, k := range CrossDomainSensitivity(master) {
		fmt.Fprintf(b, "| %s(%s)", k.Knob, k.Domain)
		for _, c := range k.Cells {
			if c.PctChange == 0 {
				fmt.Fprintf(b, " | -")
			} else {
				fmt.Fprintf(b, " | %+.1f%%", c.PctChange)
			}
		}
		fmt.Fprintf(b, " |\n")
	}
	fmt.Fprintf(b, "\n")
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
		fmt.Fprintf(b, "| +10%%↓ \\ 補償→")
		for _, name := range mx.Knobs {
			fmt.Fprintf(b, " | %s", name)
		}
		fmt.Fprintf(b, " |\n|---")
		for range mx.Knobs {
			fmt.Fprintf(b, "|---:")
		}
		fmt.Fprintf(b, "|\n")
		for _, row := range mx.Rows {
			fmt.Fprintf(b, "| %s", row.Knob)
			for _, c := range row.Cells {
				switch {
				case c.Compensator == row.Knob:
					fmt.Fprintf(b, " | -")
				case !c.OK:
					fmt.Fprintf(b, " | ×")
				default:
					fmt.Fprintf(b, " | %+.1f%%", c.PctChange)
				}
			}
			fmt.Fprintf(b, " |\n")
		}
		fmt.Fprintf(b, "\n")
	}
}
