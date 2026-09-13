package balance

import (
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ExpectedDamagePerAttack は1回の攻撃で与える期待ダメージを閉形式で返す。乱数を使わず、
// combat.go の rollAttack と同じ式から期待値を計算する。命中判定、クリティカル、ダイス1-6、
// 防御差し引きの下限保証まで含む。ダイスと防御の max が非線形なのでダイス6面を列挙して厳密化する。
func ExpectedDamagePerAttack(attacker, defender CombatantStats, weapon WeaponStats) float64 {
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
		base := baseAbil + die + weapon.Damage
		normal := max(base-defender.Defense, formula.MinDamage)
		crit := max(formula.ApplyCritical(base)-defender.Defense, formula.MinDamage)
		sum += pCrit*float64(crit) + pNormalHit*float64(normal)
	}
	return sum / float64(formula.DamageRandomRange)
}

// ExpectedTTK は attacker が defender を倒すのに要する撃破所要打数を返す。HP を1撃期待ダメージで
// 割った定義で、最後の一撃のオーバーキルは無視する。停止時刻の期待値とは別物で、こちらは
// 難易度の比較指標として素直な連続近似になる。期待ダメージがゼロなら倒せないので 0 を返す。
func ExpectedTTK(attacker, defender CombatantStats, weapon WeaponStats) float64 {
	dmg := ExpectedDamagePerAttack(attacker, defender, weapon)
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

// DifficultyCurve は経過日 1..days の序盤戦闘難易度を返す。dangerLevel(day) から
// 敵テーブルの該当帯を重みで期待し、期待撃破ターンとその比を日ごとに評価する。乱数を使わない。
func DifficultyCurve(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days int) ([]DayMetric, error) {
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
			enemyWeapon, err := LoadEnemyWeapon(master, entry.Id)
			if err != nil {
				enemyWeapon = WeaponStats{}
			}
			w := entry.Weight
			wSum += w
			playerTTKSum += w * ExpectedTTK(player, enemy, playerWeapon)
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
