package balance

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ExpectedDamagePerAttack は1回の攻撃で与える期待ダメージを閉形式で返す。熟練度倍率のない静的下限で、
// スキルを織り込むときは ExpectedDamagePerAttackWithSkill を使う。
func ExpectedDamagePerAttack(attacker, defender CombatantStats, weapon WeaponStats) float64 {
	return ExpectedDamagePerAttackWithSkill(attacker, defender, weapon, consts.PercentBase)
}

// ExpectedDamagePerAttackWithSkill は熟練度倍率 skillMult を織り込んだ1回の期待ダメージを返す。calculateDamage と
// 同じく base 全体へ倍率を切り捨てで掛け、クリティカル、防御下限の順で計算する。ダイスと防御の max が非線形なので
// ダイス6面を列挙して厳密化する。skillMult が PercentBase なら静的下限に一致する。
func ExpectedDamagePerAttackWithSkill(attacker, defender CombatantStats, weapon WeaponStats, skillMult consts.Percent) float64 {
	// CalcHitRate は MinHitRate 以上へクランプするので hitRate は 0 にならない。よって命中確率と
	// max(...,1) の下限から期待ダメージは常に正になり、ExpectedTTK もゼロ除算や 0 に落ちない。
	hitRate := formula.CalcHitRate(attacker.Dexterity, defender.Agility, weapon.Accuracy)

	baseAbil := attacker.Strength
	if weapon.IsRanged {
		baseAbil = attacker.Sensation
	}

	// roll 1..DiceMax のうち、1..min(crit,hitRate) がクリティカル、その先 hitRate まで通常命中。
	critRolls := min(hitRate, formula.CriticalHitThreshold)
	pCrit := float64(critRolls) / float64(formula.DiceMax)
	pNormalHit := float64(hitRate-critRolls) / float64(formula.DiceMax)

	sum := 0.0
	for die := 1; die <= formula.DamageRandomRange; die++ {
		base := skillMult.ApplyInt(baseAbil + die + weapon.Damage)
		normal := max(base-defender.Defense, formula.MinDamage)
		crit := max(formula.ApplyCritical(base)-defender.Defense, formula.MinDamage)
		sum += pCrit*float64(crit) + pNormalHit*float64(normal)
	}
	return sum / float64(formula.DamageRandomRange)
}

// ExpectedTTK は attacker が defender を倒すのに要する撃破所要打数を返す。HP を1撃期待ダメージで割った連続近似で、
// オーバーキルは無視する。難易度の比較指標。期待ダメージ0なら倒せず 0 を返す。
func ExpectedTTK(attacker, defender CombatantStats, weapon WeaponStats) float64 {
	return ExpectedTTKWithSkill(attacker, defender, weapon, consts.PercentBase)
}

// ExpectedTTKWithSkill は熟練度倍率 skillMult を織り込んだ撃破所要打数を返す。スキルの手応えを
// 撃破速度で見るときに使う。
func ExpectedTTKWithSkill(attacker, defender CombatantStats, weapon WeaponStats, skillMult consts.Percent) float64 {
	dmg := ExpectedDamagePerAttackWithSkill(attacker, defender, weapon, skillMult)
	if dmg <= 0 {
		return 0
	}
	return float64(defender.HP) / dmg
}

// DayMetric は経過日1日ぶんの序盤戦闘の難易度指標。危険度から敵プールを引き、
// プレイヤーと敵の期待撃破ターンを重み付き期待で評価する。
type DayMetric struct {
	Day        int     // 経過日数
	Danger     int     // その日の危険度
	PlayerTTK  float64 // プレイヤーが敵を倒す期待ターン。小さいほど有利
	EnemyTTK   float64 // 敵がプレイヤーを倒す期待ターン。大きいほど安全
	PowerRatio float64 // 戦力比 EnemyTTK / PlayerTTK。1超で有利、1近傍で拮抗、1未満で劣勢
}

// DifficultyCurve は経過日 1..days の序盤戦闘難易度を返す。dangerLevel(day) から敵テーブルの該当帯を重みで期待し、
// 期待撃破ターンと比を評価する。静的下限で見る。スキル込みは DifficultyCurveWithSkill。乱数なし。
func DifficultyCurve(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days int) ([]DayMetric, error) {
	return DifficultyCurveWithSkill(master, player, playerWeapon, enemyTableName, days, consts.PercentBase)
}

// DifficultyCurveWithSkill は熟練度倍率 skillMult を織り込んだ難易度カーブを返す。倍率はプレイヤー側だけに効き、
// スキル成長ぶんを実ゲームと同じ base 全体への切り捨てで反映する。
func DifficultyCurveWithSkill(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days int, skillMult consts.Percent) ([]DayMetric, error) {
	table, err := raw.GetEnemyTable(master, enemyTableName)
	if err != nil {
		return nil, err
	}

	out := make([]DayMetric, 0, days)
	for day := 1; day <= days; day++ {
		danger := query.DangerLevelForDay(day)

		var wSum, playerTTKSum, enemyTTKSum float64
		for _, entry := range table.Entries {
			if danger < entry.MinDanger || danger > entry.MaxDanger {
				continue
			}
			enemy, err := LoadCombatantFromMember(master, entry.Id)
			if err != nil {
				return nil, err
			}
			// LoadEnemyWeapon は武器を持たない敵を内部で素手へフォールバックする。それでもエラーなら
			// 本物のデータ異常なので握り潰さず伝播する。LoadCombatantFromMember と扱いを揃える。
			enemyWeapon, err := LoadEnemyWeapon(master, entry.Id)
			if err != nil {
				return nil, err
			}
			w := entry.Weight
			wSum += w
			playerTTKSum += w * ExpectedTTKWithSkill(player, enemy, playerWeapon, skillMult)
			enemyTTKSum += w * ExpectedTTK(enemy, player, enemyWeapon)
		}
		if wSum == 0 {
			// この危険度に敵がいない帯。カーブに穴を残さないよう空の指標を置く
			out = append(out, DayMetric{Day: day, Danger: danger})
			continue
		}

		playerTTK := playerTTKSum / wSum
		enemyTTK := enemyTTKSum / wSum
		powerRatio := 0.0
		if playerTTK > 0 {
			powerRatio = enemyTTK / playerTTK
		}
		out = append(out, DayMetric{Day: day, Danger: danger, PlayerTTK: playerTTK, EnemyTTK: enemyTTK, PowerRatio: powerRatio})
	}
	return out, nil
}
