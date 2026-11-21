package balance

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
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
func LoadWeaponFromItem(master rawschema.Raws, itemID string) (WeaponStats, error) {
	spec, err := raw.NewItemSpec(master, itemID)
	if err != nil {
		return WeaponStats{}, fmt.Errorf("failed to load weapon %q: %w", itemID, err)
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
		return meleeWeapon(spec.Melee), nil
	}

	return WeaponStats{}, fmt.Errorf("item %q has neither Melee nor Fire", itemID)
}

// LoadWeaponFromBodyAttack はraw.Masterの体の攻撃からWeaponStatsを生成する
func LoadWeaponFromBodyAttack(master rawschema.Raws, bodyAttackID string) (WeaponStats, error) {
	melee, _, err := raw.BodyAttackMelee(master, bodyAttackID)
	if err != nil {
		return WeaponStats{}, fmt.Errorf("failed to load body attack %q: %w", bodyAttackID, err)
	}
	return meleeWeapon(melee), nil
}

// meleeWeapon は近接攻撃の WeaponStats を返す
func meleeWeapon(melee *gc.Melee) WeaponStats {
	return WeaponStats{Damage: melee.Damage, Accuracy: melee.Accuracy}
}

// LoadWeaponFromMember は member が攻撃に使う武器の WeaponStats を返す。CommandTable の最も重みの高い行を代表に取り、
// CommandTable が無ければ素手にする。コマンドの重み分布は反映しない簡略化で、DifficultyCurve の敵プール重みとは
// 別の近似なので二重の簡略化になる点に注意する。
func LoadWeaponFromMember(master rawschema.Raws, memberID string) (WeaponStats, error) {
	member, err := raw.FindMember(master, memberID)
	if err != nil {
		return WeaponStats{}, err
	}

	if member.CommandTableId == nil {
		// CommandTableがない場合は素手
		return LoadWeaponFromBodyAttack(master, raw.BareHands)
	}

	ct, err := raw.GetCommandTable(master, *member.CommandTableId)
	if err != nil {
		return WeaponStats{}, err
	}

	if len(ct.Entries) == 0 {
		return LoadWeaponFromBodyAttack(master, raw.BareHands)
	}

	// 最も重みの高い攻撃を代表として使う
	bestEntry := ct.Entries[0]
	for _, entry := range ct.Entries[1:] {
		if entry.Weight > bestEntry.Weight {
			bestEntry = entry
		}
	}

	melee, _, err := raw.CommandEntryMelee(master, bestEntry)
	if err != nil {
		return WeaponStats{}, fmt.Errorf("command table %q: %w", ct.Id, err)
	}
	return meleeWeapon(melee), nil
}
