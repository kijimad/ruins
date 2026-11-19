package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpectedDamagePerAttack_手計算と一致(t *testing.T) {
	t.Parallel()
	// 攻撃側 STR=5 DEX=5、素手相当 dmg=3 acc=100、防御側 AGI=5 DEF=3。
	// 命中率は 80+(5-5)*2+(100-80)=100 → clamp 95。クリティカルは roll<=5 なので pCrit=0.05、
	// 通常命中 pNormal=0.90。ダイス1-6を列挙して手計算すると期待ダメージは 8.35。
	attacker := CombatantStats{Strength: 5, Sensation: 5, Dexterity: 5, Agility: 5}
	defender := CombatantStats{Agility: 5, Defense: 3}
	weapon := WeaponStats{Damage: 3, Accuracy: 100}

	got := ExpectedDamagePerAttack(attacker, defender, weapon)
	assert.InDelta(t, 8.35, got, 1e-9)
}

func TestExpectedTTK_期待ダメージで割る(t *testing.T) {
	t.Parallel()
	// ExpectedTTK は HP を1撃期待ダメージで割った実効撃破打数と定義する。オーバーキルは無視する。
	attacker := CombatantStats{Strength: 5, Sensation: 5, Dexterity: 5, Agility: 5}
	defender := CombatantStats{HP: 80, Agility: 5, Defense: 3}
	weapon := WeaponStats{Damage: 3, Accuracy: 100}

	dmg := ExpectedDamagePerAttack(attacker, defender, weapon)
	assert.InDelta(t, 80.0/dmg, ExpectedTTK(attacker, defender, weapon), 1e-9)
}
