package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/query"
)

// DefaultAttacksPerDay は想定プレイヤーが1日に積む攻撃回数の仮説。累積攻撃回数からスキル値を導く。
// 80×20日≈1600攻撃で素手スキルはおよそ Lv30 相当。行動依存なので見直す前提の値。
const DefaultAttacksPerDay = 80

// 想定装備トラックの仮説。能力は成長させず装備で生存力を伸ばす方針で、進行に沿って装備由来の防御が増える。
// 防御は per-hit を緩和し、床には即死級の敵ダメージを想定では多ターン消耗に変える。見直す前提の値。
// Params のつまみでなく定数なので、感度・交換レートの摂動対象外。動かした影響を分析機械では測れない。
const (
	DefaultDefensePerDay   = 0.8
	ExpectedGearDefenseCap = 16
)

// ExpectedSkillLevelAtDay は attacksPerDay の頻度で day 日戦い続けた想定プレイヤーの到達スキル値を返す。
// 攻撃回数からスキル値への写像は実物の skill.GainExp 反復を通す。
func ExpectedSkillLevelAtDay(abilityValue, attacksPerDay, day int) int {
	return SkillLevelAfterAttacks(abilityValue, attacksPerDay*day)
}

// ExpectedGearDefenseAtDay は day 日時点の想定プレイヤーの装備由来防御を返す。1日あたり
// DefaultDefensePerDay ずつ増え ExpectedGearDefenseCap で頭打ちになる。基礎能力の防御には足す差分。
func ExpectedGearDefenseAtDay(day int) int {
	d := min(int(math.Round(DefaultDefensePerDay*float64(day))), ExpectedGearDefenseCap)
	return d
}

// ProgressionDay は経過日1日ぶんの、静的下限と想定プレイヤーの戦闘リスクを並べた進行の断面。下限は
// スキル0・無装備の最悪ケース、想定はその日までに育ったスキルと装備防御を織り込んだ体験。両者の帯が
// 進行度調整の視界になる。
type ProgressionDay struct {
	Day           int
	Danger        int
	SkillLevel    int     // その日の想定スキル値
	GearDefense   int     // その日の想定装備防御
	DeathFloor    float64 // スキル0・無装備の死亡確率
	DeathExpected float64 // 想定スキル・想定装備の死亡確率
	TurnsFloor    float64 // 下限の期待決着ターン
	TurnsExpected float64 // 想定の期待決着ターン
}

// ProgressionCurve は経過日 1..days の静的下限と想定プレイヤーの戦闘リスクを並べて返す。難易度は日→危険度、プレイヤーは
// attacksPerDay からその日の想定スキルで進む。下限と想定の帯で体験曲線を見る。乱数を使わない。
func ProgressionCurve(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days, attacksPerDay int) ([]ProgressionDay, error) {
	out := make([]ProgressionDay, 0, days)
	for day := 1; day <= days; day++ {
		danger := query.DangerLevelForDay(day)
		level := ExpectedSkillLevelAtDay(0, attacksPerDay, day)
		mult := SkillDamagePercent(level)
		gearDef := ExpectedGearDefenseAtDay(day)

		floorDeath, floorTurns, ok, err := PoolCombatRisk(master, player, playerWeapon, enemyTableName, danger, consts.PercentBase)
		if err != nil {
			return nil, err
		}
		if !ok {
			out = append(out, ProgressionDay{Day: day, Danger: danger, SkillLevel: level, GearDefense: gearDef})
			continue
		}
		// 想定プレイヤーは基礎能力に装備防御を足す。防御が per-hit を緩和し、生存力の器になる。
		expPlayer := player
		expPlayer.Defense += gearDef
		expDeath, expTurns, _, err := PoolCombatRisk(master, expPlayer, playerWeapon, enemyTableName, danger, mult)
		if err != nil {
			return nil, err
		}
		out = append(out, ProgressionDay{
			Day: day, Danger: danger, SkillLevel: level, GearDefense: gearDef,
			DeathFloor: floorDeath, DeathExpected: expDeath,
			TurnsFloor: floorTurns, TurnsExpected: expTurns,
		})
	}
	return out, nil
}
