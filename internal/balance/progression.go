package balance

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/query"
)

// DefaultAttacksPerDay は想定プレイヤーが1日に積む攻撃回数の仮説。行動依存量なので設計仮説として置く。
// これに日数を掛けた累積攻撃回数からスキル値を導く。80×20日で約1600攻撃、素手スキルはおよそ Lv30 相当に
// 達し、AttacksToSkillLevel(0,30)=1626 とおおむね整合する。見直す前提の値。
const DefaultAttacksPerDay = 80

// ExpectedSkillLevelAtDay は attacksPerDay の頻度で day 日戦い続けた想定プレイヤーの到達スキル値を返す。
// 攻撃回数からスキル値への写像は実物の skill.GainExp 反復を通す。
func ExpectedSkillLevelAtDay(abilityValue, attacksPerDay, day int) int {
	return SkillLevelAfterAttacks(abilityValue, attacksPerDay*day)
}

// ProgressionDay は経過日1日ぶんの、静的下限と想定プレイヤーの戦闘リスクを並べた進行の断面。下限は
// スキル0の最悪ケース、想定はその日までに育ったスキルを織り込んだ体験。両者の帯が進行度調整の視界になる。
type ProgressionDay struct {
	Day           int
	Danger        int
	SkillLevel    int     // その日の想定スキル値
	DeathFloor    float64 // スキル0の死亡確率
	DeathExpected float64 // 想定スキルの死亡確率
	TurnsFloor    float64 // スキル0の期待決着ターン
	TurnsExpected float64 // 想定スキルの期待決着ターン
}

// ProgressionCurve は経過日 1..days について、静的下限と想定プレイヤーの戦闘リスクを並べて返す。
// 難易度の側は日→危険度で進み、プレイヤーの側は attacksPerDay からその日の想定スキルで進む。両者を
// 同じ日軸で突き合わせることで、下限だけでなく普通に育ったプレイヤーの体験曲線を見る。乱数を使わない。
func ProgressionCurve(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, days, attacksPerDay int) ([]ProgressionDay, error) {
	out := make([]ProgressionDay, 0, days)
	for day := 1; day <= days; day++ {
		danger := query.DangerLevelForDay(day)
		level := ExpectedSkillLevelAtDay(0, attacksPerDay, day)
		mult := SkillDamagePercent(level)

		floorDeath, floorTurns, ok, err := PoolCombatRisk(master, player, playerWeapon, enemyTableName, danger, consts.PercentBase)
		if err != nil {
			return nil, err
		}
		if !ok {
			out = append(out, ProgressionDay{Day: day, Danger: danger, SkillLevel: level})
			continue
		}
		expDeath, expTurns, _, err := PoolCombatRisk(master, player, playerWeapon, enemyTableName, danger, mult)
		if err != nil {
			return nil, err
		}
		out = append(out, ProgressionDay{
			Day: day, Danger: danger, SkillLevel: level,
			DeathFloor: floorDeath, DeathExpected: expDeath,
			TurnsFloor: floorTurns, TurnsExpected: expTurns,
		})
	}
	return out, nil
}
