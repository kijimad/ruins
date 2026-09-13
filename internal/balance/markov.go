package balance

import (
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// combatAttackCap は撃破攻撃数分布を打ち切る上限。命中率が低いほど裾が長く伸びるので大きめに取る。
// 生存質量が無視できるまで下の DP は早期終了するため、通常はこの上限に達しない。
const combatAttackCap = 4000

// CombatOutcome は1対1戦闘を吸収マルコフ連鎖として厳密に解いた結果。期待値でなく分布を持つので、
// 平均では見えない突然死の裾を捉える。ExpectedTTK が捨てていた分散側の情報。
type CombatOutcome struct {
	PlayerDeathProb float64 // プレイヤーが倒される確率。P(敵の撃破攻撃数 < 自分の撃破攻撃数)
	ExpectedTurns   float64 // 決着までの期待ターン数。min(自分の撃破攻撃数, 敵の撃破攻撃数)の期待値
	TTKMedian       int     // 決着ターンの中央値
	TTKp95          int     // 決着ターンの95パーセンタイル。長引く不運側の裾
}

// attackDamagePMF は1回の攻撃が与えるダメージの確率分布を返す。添字がダメージ量、値が確率で、
// 添字0は命中しなかった場合。combat.go の rollAttack と同じ命中判定・クリティカル・ダイス1-6・
// 防御下限をそのまま確率へ写す。乱数を使わずに rollAttack の分布を厳密に再現する。
func attackDamagePMF(attacker, defender CombatantStats, weapon WeaponStats) map[int]float64 {
	hitRate := formula.CalcHitRate(attacker.Dexterity, defender.Agility, weapon.Accuracy)
	baseAbil := attacker.Strength
	if weapon.IsRanged {
		baseAbil = attacker.Sensation
	}
	critRolls := min(hitRate, formula.CriticalHitThreshold)
	pCrit := float64(critRolls) / float64(formula.DiceMax)
	pNormal := float64(hitRate-critRolls) / float64(formula.DiceMax)
	pMiss := float64(formula.DiceMax-hitRate) / float64(formula.DiceMax)

	pmf := make(map[int]float64, 2*formula.DamageRandomRange+1)
	pmf[0] += pMiss
	for die := 1; die <= formula.DamageRandomRange; die++ {
		base := baseAbil + die + weapon.Damage
		normal := max(base-defender.Defense, formula.MinDamage)
		crit := max(formula.ApplyCritical(base)-defender.Defense, formula.MinDamage)
		pmf[normal] += pNormal / float64(formula.DamageRandomRange)
		pmf[crit] += pCrit / float64(formula.DamageRandomRange)
	}
	return pmf
}

// killDistribution は attacker がダメージ分布 pmf で HP hp の defender を倒すのに要する攻撃回数 k の
// 確率分布を返す。添字が攻撃回数で kill[k]=P(ちょうど k 回目で倒す)。残 HP を状態とする1次元 DP で、
// 各攻撃は生存質量を減らしていく。生存質量が無視できるまで早期終了する。
func killDistribution(pmf map[int]float64, hp int) []float64 {
	kill := make([]float64, combatAttackCap+1)
	if hp <= 0 {
		kill[1] = 1
		return kill
	}
	alive := make([]float64, hp+1)
	alive[hp] = 1
	for k := 1; k <= combatAttackCap; k++ {
		next := make([]float64, hp+1)
		var killed, aliveMass float64
		for r := 1; r <= hp; r++ {
			if alive[r] == 0 {
				continue
			}
			for d, p := range pmf {
				if p == 0 {
					continue
				}
				if d >= r {
					killed += alive[r] * p
				} else {
					next[r-d] += alive[r] * p
					aliveMass += alive[r] * p
				}
			}
		}
		kill[k] = killed
		alive = next
		if aliveMass < 1e-12 {
			break
		}
	}
	return kill
}

// CombatDistribution は1対1戦闘の結果分布を厳密に解く。プレイヤー先攻の交互攻撃なので、自分の撃破
// 攻撃数 Kp と敵の撃破攻撃数 Ke は独立で、勝敗は Kp<=Ke で決まる。よって死亡確率は P(Ke<Kp)、決着
// ターンは min(Kp,Ke) の分布として畳み込みで求まる。
func CombatDistribution(player, enemy CombatantStats, playerWeapon, enemyWeapon WeaponStats) CombatOutcome {
	kp := killDistribution(attackDamagePMF(player, enemy, playerWeapon), enemy.HP)
	ke := killDistribution(attackDamagePMF(enemy, player, enemyWeapon), player.HP)

	// 累積分布。cumKp[k]=P(Kp<=k)
	cumKp := make([]float64, len(kp))
	cumKe := make([]float64, len(ke))
	var accP, accE float64
	for k := 1; k < len(kp); k++ {
		accP += kp[k]
		cumKp[k] = accP
	}
	for k := 1; k < len(ke); k++ {
		accE += ke[k]
		cumKe[k] = accE
	}

	// 死亡確率 P(Ke<Kp) = Σ ke[k]·P(Kp>k)
	var death float64
	for k := 1; k < len(ke); k++ {
		death += ke[k] * (1 - cumKp[k])
	}

	// 決着ターン min(Kp,Ke) の分布。P(min=t)=P(Kp=t)P(Ke>=t)+P(Ke=t)P(Kp>=t)-P(Kp=t)P(Ke=t)
	var expTurns, cumMin float64
	median, p95 := 0, 0
	for t := 1; t < len(kp); t++ {
		pKpGeT := 1 - cumKp[t-1]
		pKeGeT := 1 - cumKe[t-1]
		pMin := kp[t]*pKeGeT + ke[t]*pKpGeT - kp[t]*ke[t]
		if pMin <= 0 {
			continue
		}
		expTurns += float64(t) * pMin
		cumMin += pMin
		if median == 0 && cumMin >= 0.5 {
			median = t
		}
		if p95 == 0 && cumMin >= 0.95 {
			p95 = t
		}
	}
	return CombatOutcome{PlayerDeathProb: death, ExpectedTurns: expTurns, TTKMedian: median, TTKp95: p95}
}

// DayRisk は経過日1日ぶんの戦闘リスク。敵プールを重みで期待した死亡確率と決着ターンを持つ。
// DifficultyCurve の PowerRatio が期待値の比なのに対し、こちらは分布から突然死の確率を直接測る。
type DayRisk struct {
	Day       int
	Danger    int
	DeathProb float64 // その日の敵プールに1体遭遇したときの重み付き死亡確率
	ExpTurns  float64 // 重み付き期待決着ターン
}

// CombatRiskCurve は経過日 1..days の戦闘リスクを敵プールの重みで期待して返す。各敵との1対1を
// マルコフ連鎖で解き、出現重みで平均する。乱数を使わない。
func CombatRiskCurve(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days int) ([]DayRisk, error) {
	table, err := raw.GetEnemyTable(master, enemyTableName)
	if err != nil {
		return nil, err
	}
	out := make([]DayRisk, 0, days)
	for day := 1; day <= days; day++ {
		danger := query.DangerLevelForDay(day)
		var wSum, deathSum, turnSum float64
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
				return nil, err
			}
			o := CombatDistribution(player, enemy, playerWeapon, enemyWeapon)
			w := entry.Weight
			wSum += w
			deathSum += w * o.PlayerDeathProb
			turnSum += w * o.ExpectedTurns
		}
		if wSum == 0 {
			out = append(out, DayRisk{Day: day, Danger: danger})
			continue
		}
		out = append(out, DayRisk{Day: day, Danger: danger, DeathProb: deathSum / wSum, ExpTurns: turnSum / wSum})
	}
	return out, nil
}
