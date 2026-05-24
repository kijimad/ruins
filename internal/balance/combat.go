package balance

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/raw"
)

// CombatantStats は戦闘シミュレーション用のステータス
type CombatantStats struct {
	HP        int
	Strength  int
	Sensation int
	Dexterity int
	Agility   int
	Defense   int
}

// WeaponStats は武器のシミュレーション用パラメータ
type WeaponStats struct {
	Damage   int
	Accuracy int
	IsRanged bool
}

// BattleResult は1戦闘の結果
type BattleResult struct {
	Turns       int
	PlayerWon   bool
	DamageTaken int
}

// rollAttack は1回の攻撃をロールし、与えたダメージを返す。
// 本体（activity/attack.go）と同じ計算順序: 基本ダメージ → クリティカル → 防御減算 → クランプ
func rollAttack(attacker, defender CombatantStats, weapon WeaponStats, rng *rand.Rand) int {
	hitRate := formula.CalcHitRate(attacker.Dexterity, defender.Agility, weapon.Accuracy)
	roll := rng.IntN(formula.DiceMax) + 1
	if roll > hitRate {
		return 0
	}

	baseAbil := attacker.Strength
	if weapon.IsRanged {
		baseAbil = attacker.Sensation
	}
	baseDamage := baseAbil + rng.IntN(formula.DamageRandomRange) + 1 + weapon.Damage
	if roll <= formula.CriticalHitThreshold {
		baseDamage = formula.ApplyCritical(baseDamage)
	}

	finalDamage := baseDamage - defender.Defense
	if finalDamage < formula.MinDamage {
		finalDamage = formula.MinDamage
	}
	return finalDamage
}

// SimulateBattle は1戦闘を模擬する
func SimulateBattle(player, enemy CombatantStats, playerWeapon, enemyWeapon WeaponStats, rng *rand.Rand) BattleResult {
	playerHP := player.HP
	enemyHP := enemy.HP
	turns := 0
	damageTaken := 0

	for playerHP > 0 && enemyHP > 0 {
		turns++

		enemyHP -= rollAttack(player, enemy, playerWeapon, rng)
		if enemyHP <= 0 {
			break
		}

		dmg := rollAttack(enemy, player, enemyWeapon, rng)
		playerHP -= dmg
		damageTaken += dmg
	}

	return BattleResult{
		Turns:       turns,
		PlayerWon:   playerHP > 0,
		DamageTaken: damageTaken,
	}
}

// LoadCombatantFromMember はraw.Masterのメンバー定義からCombatantStatsを生成する
func LoadCombatantFromMember(master *raw.Master, name string) (CombatantStats, error) {
	spec, err := master.NewMemberSpec(name)
	if err != nil {
		return CombatantStats{}, fmt.Errorf("メンバー %q のロードに失敗: %w", name, err)
	}
	if spec.Abilities == nil {
		return CombatantStats{}, fmt.Errorf("メンバー %q にAbilitiesがありません", name)
	}

	abils := spec.Abilities
	stats := CombatantStats{
		Strength:  abils.Strength.Base,
		Sensation: abils.Sensation.Base,
		Dexterity: abils.Dexterity.Base,
		Agility:   abils.Agility.Base,
		Defense:   abils.Defense.Base,
	}
	stats.HP = formula.CalcHP(abils.Vitality.Base, abils.Strength.Base, abils.Sensation.Base)

	return stats, nil
}

// LoadWeaponFromItem はraw.MasterのアイテムからWeaponStatsを生成する
func LoadWeaponFromItem(master *raw.Master, name string) (WeaponStats, error) {
	spec, err := master.NewItemSpec(name)
	if err != nil {
		return WeaponStats{}, fmt.Errorf("武器 %q のロードに失敗: %w", name, err)
	}

	if spec.Melee != nil {
		return WeaponStats{
			Damage:   spec.Melee.Damage,
			Accuracy: spec.Melee.Accuracy,
			IsRanged: spec.Melee.AttackCategory.Range == gc.AttackRangeRanged,
		}, nil
	}
	if spec.Fire != nil {
		return WeaponStats{
			Damage:   spec.Fire.Damage,
			Accuracy: spec.Fire.Accuracy,
			IsRanged: true,
		}, nil
	}

	return WeaponStats{}, fmt.Errorf("アイテム %q にMeleeもFireもありません", name)
}

// LoadEnemyWeapon は敵のCommandTableから武器を取得しWeaponStatsを返す
func LoadEnemyWeapon(master *raw.Master, enemyName string) (WeaponStats, error) {
	spec, err := master.NewMemberSpec(enemyName)
	if err != nil {
		return WeaponStats{}, err
	}

	if spec.CommandTable == nil {
		// CommandTableがない場合は素手
		return LoadWeaponFromItem(master, "素手")
	}

	ct, err := master.GetCommandTable(spec.CommandTable.Name)
	if err != nil {
		return WeaponStats{}, err
	}

	if len(ct.Entries) == 0 {
		return LoadWeaponFromItem(master, "素手")
	}

	// 最も重みの高い攻撃を代表として使う
	bestEntry := ct.Entries[0]
	for _, entry := range ct.Entries[1:] {
		if entry.Weight > bestEntry.Weight {
			bestEntry = entry
		}
	}

	return LoadWeaponFromItem(master, bestEntry.Weapon)
}
