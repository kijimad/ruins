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

// LoadEnemyWeapon は敵の CommandTable から最も重みの高い武器を代表に取り WeaponStats を返す。コマンドの重み分布は
// 反映しない簡略化で、DifficultyCurve の敵プール重みとは別の近似なので二重の簡略化になる点に注意する。
func LoadEnemyWeapon(master rawschema.Raws, enemyName string) (WeaponStats, error) {
	member, err := raw.FindMember(master, enemyName)
	if err != nil {
		return WeaponStats{}, err
	}

	if member.CommandTableId == nil {
		// CommandTableがない場合は素手
		return LoadWeaponFromItem(master, "bare_hands")
	}

	ct, err := raw.GetCommandTable(master, *member.CommandTableId)
	if err != nil {
		return WeaponStats{}, err
	}

	if len(ct.Entries) == 0 {
		return LoadWeaponFromItem(master, "bare_hands")
	}

	// 最も重みの高い攻撃を代表として使う
	bestEntry := ct.Entries[0]
	for _, entry := range ct.Entries[1:] {
		if entry.Weight > bestEntry.Weight {
			bestEntry = entry
		}
	}

	return LoadWeaponFromItem(master, bestEntry.Id)
}
