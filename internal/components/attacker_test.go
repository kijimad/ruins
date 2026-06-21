package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMelee_AttackerInterface(t *testing.T) {
	t.Parallel()

	m := &Melee{
		Accuracy:       80,
		Damage:         25,
		AttackCount:    2,
		Element:        ElementTypeFire,
		AttackCategory: AttackSword,
		Cost:           100,
		TargetType:     TargetType{TargetGroup: TargetGroupEnemy, TargetNum: TargetSingle},
	}

	var atk Attacker = m
	assert.Equal(t, 80, atk.GetAccuracy())
	assert.Equal(t, 25, atk.GetDamage())
	assert.Equal(t, 2, atk.GetAttackCount())
	assert.Equal(t, ElementTypeFire, atk.GetElement())
	assert.Equal(t, AttackSword, atk.GetAttackCategory())
	assert.Equal(t, 100, atk.GetCost())
	assert.Equal(t, TargetGroupEnemy, atk.GetTargetType().TargetGroup)
	assert.Equal(t, TargetSingle, atk.GetTargetType().TargetNum)
}

func TestFire_AttackerInterface(t *testing.T) {
	t.Parallel()

	f := &Fire{
		Accuracy:       70,
		Damage:         30,
		AttackCount:    1,
		Element:        ElementTypeThunder,
		AttackCategory: AttackRifle,
		Cost:           150,
		TargetType:     TargetType{TargetGroup: TargetGroupEnemy, TargetNum: TargetAll},
	}

	var atk Attacker = f
	assert.Equal(t, 70, atk.GetAccuracy())
	assert.Equal(t, 30, atk.GetDamage())
	assert.Equal(t, 1, atk.GetAttackCount())
	assert.Equal(t, ElementTypeThunder, atk.GetElement())
	assert.Equal(t, AttackRifle, atk.GetAttackCategory())
	assert.Equal(t, 150, atk.GetCost())
	assert.Equal(t, TargetAll, atk.GetTargetType().TargetNum)
}
