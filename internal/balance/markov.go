package balance

import (
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
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

// damageProb はダメージ量とその確率の組。攻撃1回のダメージ分布をダメージ量の昇順スライスで持つ。
// map で持つと反復順がランダムになり、浮動小数の加算が非結合なため撃破確率が実行ごとに微差を持つ。
// 昇順スライスにして加算順を固定し、結果を決定論的にする。
type damageProb struct {
	dmg int
	p   float64
}

// attackDamagePMF は1回の攻撃が与えるダメージの確率分布を、ダメージ量の昇順スライスで返す。dmg=0 は
// 命中しなかった場合。activity/attack.go の calculateDamage と同じく熟練度倍率 skillMult を base 全体へ
// 切り捨てで掛け、次にクリティカル、最後に防御下限をする。乱数を使わずに分布を厳密に再現する。
// skillMult が PercentBase なら熟練度なしで、combat.go の rollAttack の分布に一致する。
func attackDamagePMF(attacker, defender CombatantStats, weapon WeaponStats, skillMult consts.Percent) []damageProb {
	hitRate := formula.CalcHitRate(attacker.Dexterity, defender.Agility, weapon.Accuracy)
	baseAbil := attacker.Strength
	if weapon.IsRanged {
		baseAbil = attacker.Sensation
	}
	critRolls := min(hitRate, formula.CriticalHitThreshold)
	pCrit := float64(critRolls) / float64(formula.DiceMax)
	pNormal := float64(hitRate-critRolls) / float64(formula.DiceMax)
	pMiss := float64(formula.DiceMax-hitRate) / float64(formula.DiceMax)

	acc := make(map[int]float64, 2*formula.DamageRandomRange+1)
	// dmg=0 は命中しなかった場合を一意に表す。命中時の normal/crit は max(..., formula.MinDamage) で
	// 下限が MinDamage>=1 なので 0 にならず、acc[0] は miss だけを集める。この不変条件が崩れると 0 キーに
	// miss と命中が混ざり分布が壊れる。TestAttackDamagePMF_命中は0ダメージにならない で担保する。
	acc[0] += pMiss
	for die := 1; die <= formula.DamageRandomRange; die++ {
		base := skillMult.ApplyInt(baseAbil + die + weapon.Damage)
		normal := max(base-defender.Defense, formula.MinDamage)
		crit := max(formula.ApplyCritical(base)-defender.Defense, formula.MinDamage)
		acc[normal] += pNormal / float64(formula.DamageRandomRange)
		acc[crit] += pCrit / float64(formula.DamageRandomRange)
	}
	dmgs := make([]int, 0, len(acc))
	for d := range acc {
		dmgs = append(dmgs, d)
	}
	sort.Ints(dmgs)
	pmf := make([]damageProb, len(dmgs))
	for i, d := range dmgs {
		pmf[i] = damageProb{dmg: d, p: acc[d]}
	}
	return pmf
}

// killDistribution は attacker がダメージ分布 pmf で HP hp の defender を倒すのに要する攻撃回数 k の
// 確率分布を返す。添字が攻撃回数で kill[k]=P(ちょうど k 回目で倒す)。残 HP を状態とする1次元 DP で、
// 各攻撃は生存質量を減らしていく。生存質量が無視できるまで早期終了する。
func killDistribution(pmf []damageProb, hp int) []float64 {
	if hp <= 0 {
		return []float64{0, 1} // 0回では倒せず、1回目で確定して倒す
	}
	// kill[k]=P(ちょうど k 回目で倒す)。到達した攻撃回数ぶんだけ append で伸ばす。生存質量が尽きれば
	// 早期終了するので、命中率が高いほど短くなり combatAttackCap 分の確保を避けられる。
	kill := []float64{0}
	// alive と next を2枚だけ確保し ping-pong で使い回す。毎攻撃ぶん確保すると累積するため。
	alive := make([]float64, hp+1)
	next := make([]float64, hp+1)
	alive[hp] = 1
	for k := 1; k <= combatAttackCap; k++ {
		clear(next)
		var killed, aliveMass float64
		for r := 1; r <= hp; r++ {
			if alive[r] == 0 {
				continue
			}
			for _, dp := range pmf {
				if dp.p == 0 {
					continue
				}
				if dp.dmg >= r {
					killed += alive[r] * dp.p
				} else {
					next[r-dp.dmg] += alive[r] * dp.p
					aliveMass += alive[r] * dp.p
				}
			}
		}
		kill = append(kill, killed)
		alive, next = next, alive
		if aliveMass < 1e-12 {
			break
		}
	}
	return kill
}

// CombatDistribution は熟練度なしの1対1戦闘の結果分布を解く。静的下限で、スキルを織り込むときは
// CombatDistributionWithSkill を使う。
func CombatDistribution(player, enemy CombatantStats, playerWeapon, enemyWeapon WeaponStats) CombatOutcome {
	return CombatDistributionWithSkill(player, enemy, playerWeapon, enemyWeapon, consts.PercentBase)
}

// CombatDistributionWithSkill はプレイヤーの熟練度倍率 playerSkillMult を織り込んだ1対1戦闘の結果分布を
// 厳密に解く。プレイヤー先攻の交互攻撃なので、自分の撃破攻撃数 Kp と敵の撃破攻撃数 Ke は独立で、勝敗は
// Kp<=Ke で決まる。よって死亡確率は P(Ke<Kp)、決着ターンは min(Kp,Ke) の分布として畳み込みで求まる。
// 倍率はプレイヤーの攻撃にだけ効き、敵は静的下限のまま。
func CombatDistributionWithSkill(player, enemy CombatantStats, playerWeapon, enemyWeapon WeaponStats, playerSkillMult consts.Percent) CombatOutcome {
	kp := killDistribution(attackDamagePMF(player, enemy, playerWeapon, playerSkillMult), enemy.HP)
	ke := killDistribution(attackDamagePMF(enemy, player, enemyWeapon, consts.PercentBase), player.HP)

	// kp と ke は早期終了で長さが異なりうる。以降の畳み込みは相手側を同じ添字で引くので、短い方を
	// ゼロ詰めして長さを揃える。撃破質量が尽きた後の kill は0なので、詰めても分布は変わらない。
	n := max(len(kp), len(ke))
	for len(kp) < n {
		kp = append(kp, 0)
	}
	for len(ke) < n {
		ke = append(ke, 0)
	}

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

// PoolCombatRisk は危険度 danger の敵プールに1体遭遇したときの、重み付き死亡確率と期待決着ターンを
// 返す。プレイヤーの熟練度倍率 playerSkillMult を織り込む。各敵との1対1をマルコフ連鎖で解き、出現重みで
// 平均する。その帯に敵がいなければ ok=false。熟練度なしの静的下限は PercentBase を渡す。
func PoolCombatRisk(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, danger int, playerSkillMult consts.Percent) (deathProb, expTurns float64, ok bool, err error) {
	table, err := raw.GetEnemyTable(master, enemyTableName)
	if err != nil {
		return 0, 0, false, err
	}
	var wSum, deathSum, turnSum float64
	for _, entry := range table.Entries {
		if danger < entry.MinDanger || danger > entry.MaxDanger {
			continue
		}
		enemy, err := LoadCombatantFromMember(master, entry.Id)
		if err != nil {
			return 0, 0, false, err
		}
		enemyWeapon, err := LoadEnemyWeapon(master, entry.Id)
		if err != nil {
			return 0, 0, false, err
		}
		o := CombatDistributionWithSkill(player, enemy, playerWeapon, enemyWeapon, playerSkillMult)
		w := entry.Weight
		wSum += w
		deathSum += w * o.PlayerDeathProb
		turnSum += w * o.ExpectedTurns
	}
	if wSum == 0 {
		return 0, 0, false, nil
	}
	return deathSum / wSum, turnSum / wSum, true, nil
}

// CombatRiskCurve は経過日 1..days の戦闘リスクを敵プールの重みで期待して返す。プレイヤーは熟練度なしの
// 静的下限で見る。各敵との1対1をマルコフ連鎖で解き、出現重みで平均する。乱数を使わない。
func CombatRiskCurve(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days int) ([]DayRisk, error) {
	out := make([]DayRisk, 0, days)
	for day := 1; day <= days; day++ {
		danger := query.DangerLevelForDay(day)
		death, turns, ok, err := PoolCombatRisk(master, player, playerWeapon, enemyTableName, danger, consts.PercentBase)
		if err != nil {
			return nil, err
		}
		if !ok {
			out = append(out, DayRisk{Day: day, Danger: danger})
			continue
		}
		out = append(out, DayRisk{Day: day, Danger: danger, DeathProb: death, ExpTurns: turns})
	}
	return out, nil
}
