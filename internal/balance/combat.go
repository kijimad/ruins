package balance

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
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

// LoadCombatantFromMember はraw.Masterのメンバー定義からCombatantStatsを生成する
func LoadCombatantFromMember(master rawschema.Raws, name string) (CombatantStats, error) {
	spec, err := raw.NewMemberSpec(master, name)
	if err != nil {
		return CombatantStats{}, fmt.Errorf("failed to load member %q: %w", name, err)
	}
	if spec.Abilities == nil {
		return CombatantStats{}, fmt.Errorf("member %q has no Abilities", name)
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
func LoadWeaponFromItem(master rawschema.Raws, name string) (WeaponStats, error) {
	spec, err := raw.NewItemSpec(master, name)
	if err != nil {
		return WeaponStats{}, fmt.Errorf("failed to load weapon %q: %w", name, err)
	}

	// Fireを先にチェックする。MeleeとFireの両方を持つ武器は遠距離として扱う
	if spec.Fire != nil {
		return WeaponStats{
			Damage:   spec.Fire.Damage,
			Accuracy: spec.Fire.Accuracy,
			IsRanged: true,
		}, nil
	}
	if spec.Melee != nil {
		return WeaponStats{
			Damage:   spec.Melee.Damage,
			Accuracy: spec.Melee.Accuracy,
			IsRanged: false,
		}, nil
	}

	return WeaponStats{}, fmt.Errorf("item %q has neither Melee nor Fire", name)
}

// LoadWeapon は id の武器の WeaponStats を返す。素手のような体の攻撃と、剣のようなアイテムを同じつまみで比べるため、
// 体の攻撃の表を先に引き、無ければアイテムの表から引く
func LoadWeapon(master rawschema.Raws, id string) (WeaponStats, error) {
	if _, ok := raw.FindByKey(master.BodyAttacks, func(a rawschema.BodyAttack) string { return a.Id }, id); ok {
		return LoadWeaponFromBodyAttack(master, id)
	}
	return LoadWeaponFromItem(master, id)
}

// LoadWeaponFromBodyAttack はraw.Masterの体の攻撃からWeaponStatsを生成する
func LoadWeaponFromBodyAttack(master rawschema.Raws, id string) (WeaponStats, error) {
	melee, _, err := raw.BodyAttackMelee(master, id)
	if err != nil {
		return WeaponStats{}, fmt.Errorf("failed to load body attack %q: %w", id, err)
	}
	return WeaponStats{Damage: melee.Damage, Accuracy: melee.Accuracy}, nil
}

// LoadEnemyWeapon は敵の CommandTable から最も重みの高い武器を代表に取り WeaponStats を返す。コマンドの重み分布は
// 反映しない簡略化で、DifficultyCurve の敵プール重みとは別の近似なので二重の簡略化になる点に注意する。
func LoadEnemyWeapon(master rawschema.Raws, enemyName string) (WeaponStats, error) {
	member, err := raw.FindMember(master, enemyName)
	if err != nil {
		return WeaponStats{}, err
	}

	if member.CommandTableId == nil {
		// CommandTableがない場合は素手
		return LoadWeaponFromBodyAttack(master, "bare_hands")
	}

	ct, err := raw.GetCommandTable(master, *member.CommandTableId)
	if err != nil {
		return WeaponStats{}, err
	}

	if len(ct.Entries) == 0 {
		return LoadWeaponFromBodyAttack(master, "bare_hands")
	}

	// 最も重みの高い攻撃を代表として使う
	bestEntry := ct.Entries[0]
	for _, entry := range ct.Entries[1:] {
		if entry.Weight > bestEntry.Weight {
			bestEntry = entry
		}
	}

	if bestEntry.BodyAttack != nil {
		return LoadWeaponFromBodyAttack(master, *bestEntry.BodyAttack)
	}
	if bestEntry.Item == nil {
		return WeaponStats{}, fmt.Errorf("command table %q has an entry without bodyAttack or item", ct.Id)
	}
	return LoadWeaponFromItem(master, *bestEntry.Item)
}
