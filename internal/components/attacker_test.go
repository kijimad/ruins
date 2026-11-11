package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestMelee_AttackerInterface(t *testing.T) {
	t.Parallel()

	m := &Melee{
		Accuracy:       80,
		Damage:         25,
		AttackCount:    2,
		Element:        rawschema.ElementFIRE,
		AttackCategory: rawschema.AttackCategorySword,
		Cost:           100,
		TargetType:     TargetType{TargetGroup: rawschema.TargetGroupENEMY, TargetNum: rawschema.SINGLE},
	}

	var atk Attacker = m
	assert.Equal(t, 80, atk.GetAccuracy())
	assert.Equal(t, 25, atk.GetDamage())
	assert.Equal(t, 2, atk.GetAttackCount())
	assert.Equal(t, rawschema.ElementFIRE, atk.GetElement())
	assert.Equal(t, rawschema.AttackCategorySword, atk.GetAttackCategory())
	assert.Equal(t, 100, atk.GetCost())
	assert.Equal(t, rawschema.TargetGroupENEMY, atk.GetTargetType().TargetGroup)
	assert.Equal(t, rawschema.SINGLE, atk.GetTargetType().TargetNum)
}

func TestFire_AttackerInterface(t *testing.T) {
	t.Parallel()

	f := &Fire{
		Accuracy:       70,
		Damage:         30,
		AttackCount:    1,
		Element:        rawschema.ElementTHUNDER,
		AttackCategory: rawschema.AttackCategoryRifle,
		Cost:           150,
		TargetType:     TargetType{TargetGroup: rawschema.TargetGroupENEMY, TargetNum: rawschema.ALL},
	}

	var atk Attacker = f
	assert.Equal(t, 70, atk.GetAccuracy())
	assert.Equal(t, 30, atk.GetDamage())
	assert.Equal(t, 1, atk.GetAttackCount())
	assert.Equal(t, rawschema.ElementTHUNDER, atk.GetElement())
	assert.Equal(t, rawschema.AttackCategoryRifle, atk.GetAttackCategory())
	assert.Equal(t, 150, atk.GetCost())
	assert.Equal(t, rawschema.ALL, atk.GetTargetType().TargetNum)
}
