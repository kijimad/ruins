package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestMelee_AttackerInterface(t *testing.T) {
	t.Parallel()

	m := &Melee{
		Accuracy:       80,
		Damage:         25,
		AttackCount:    2,
		Element:        oapi.ElementFIRE,
		AttackCategory: AttackSword,
		Cost:           100,
		TargetType:     TargetType{TargetGroup: oapi.TargetGroupENEMY, TargetNum: oapi.SINGLE},
	}

	var atk Attacker = m
	assert.Equal(t, 80, atk.GetAccuracy())
	assert.Equal(t, 25, atk.GetDamage())
	assert.Equal(t, 2, atk.GetAttackCount())
	assert.Equal(t, oapi.ElementFIRE, atk.GetElement())
	assert.Equal(t, AttackSword, atk.GetAttackCategory())
	assert.Equal(t, 100, atk.GetCost())
	assert.Equal(t, oapi.TargetGroupENEMY, atk.GetTargetType().TargetGroup)
	assert.Equal(t, oapi.SINGLE, atk.GetTargetType().TargetNum)
}

func TestFire_AttackerInterface(t *testing.T) {
	t.Parallel()

	f := &Fire{
		Accuracy:       70,
		Damage:         30,
		AttackCount:    1,
		Element:        oapi.ElementTHUNDER,
		AttackCategory: AttackRifle,
		Cost:           150,
		TargetType:     TargetType{TargetGroup: oapi.TargetGroupENEMY, TargetNum: oapi.ALL},
	}

	var atk Attacker = f
	assert.Equal(t, 70, atk.GetAccuracy())
	assert.Equal(t, 30, atk.GetDamage())
	assert.Equal(t, 1, atk.GetAttackCount())
	assert.Equal(t, oapi.ElementTHUNDER, atk.GetElement())
	assert.Equal(t, AttackRifle, atk.GetAttackCategory())
	assert.Equal(t, 150, atk.GetCost())
	assert.Equal(t, oapi.ALL, atk.GetTargetType().TargetNum)
}
