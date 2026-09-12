package balance

import (
	"math/rand/v2"
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

func TestExpectedDamagePerAttack_モンテカルロと一致(t *testing.T) {
	t.Parallel()
	// 閉形式の ExpectedDamagePerAttack が実コードの rollAttack を正しく期待値化しているかを、
	// rollAttack を多数回まわした1撃あたりの平均ダメージと突き合わせて確認する。
	// 閉形式が実式からずれたらここで落ちる。ミス0・クリティカル・防御差し引きまで含む。
	attacker := CombatantStats{Strength: 6, Sensation: 4, Dexterity: 7, Agility: 5}
	defender := CombatantStats{HP: 40, Agility: 4, Defense: 2}
	weapon := WeaponStats{Damage: 4, Accuracy: 85}

	rng := rand.New(rand.NewPCG(1, 2))
	const trials = 1000000
	total := 0
	for range trials {
		total += rollAttack(attacker, defender, weapon, rng)
	}
	mc := float64(total) / float64(trials)
	closed := ExpectedDamagePerAttack(attacker, defender, weapon)

	assert.InEpsilon(t, mc, closed, 0.02)
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
