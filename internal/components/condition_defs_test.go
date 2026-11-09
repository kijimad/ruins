package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDefs は components のテストが使う不調の定義の表。値は raw の行と同じにする
var testDefs = NewConditionDefs(map[ConditionType]ConditionDef{
	ConditionHypothermia: {
		Name:                    "Hypothermia",
		PainPerSeverity:         6,
		BodyFuncDropPerSeverity: 10,
		BloodDropPerSeverity:    22,
		Cause:                   CauseFrozen,
	},
	ConditionFracture: {
		Name:                    "Fracture",
		PainPerSeverity:         18,
		BodyFuncDropPerSeverity: 20,
		Recovery:                RecoverAfterTend,
		RecoverPer:              1,
		Parts:                   []BodyPart{BodyPartArms, BodyPartHands, BodyPartLegs, BodyPartFeet},
	},
	ConditionLaceration: {
		Name:                    "Laceration",
		PainPerSeverity:         8,
		BodyFuncDropPerSeverity: 8,
		Recovery:                RecoverAfterTend,
		RecoverPer:              2,
		BloodDropPerSeverity:    25,
		Cause:                   CauseBloodLoss,
	},
	ConditionLiverIllness: {
		Name:                    "Liver illness",
		PainPerSeverity:         4,
		BodyFuncDropPerSeverity: 10,
		Recovery:                ProgressUntilTend,
		WorsenPer:               2,
		RecoverPer:              1,
		BloodDropPerSeverity:    25,
		Cause:                   CauseIllness,
	},
}, [BodyPartCount]float64{
	BodyPartHead:  2,
	BodyPartTorso: 4,
	BodyPartArms:  3,
	BodyPartHands: 1,
	BodyPartLegs:  3,
	BodyPartFeet:  1,
}, map[string]ConditionType{
	AttackFist.Type:  ConditionFracture,
	AttackSword.Type: ConditionLaceration,
})

func TestConditionDefs_CanAffect(t *testing.T) {
	t.Parallel()

	var fracture []BodyPart
	var laceration []BodyPart
	for p := range BodyPartCount {
		if testDefs.CanAffect(ConditionFracture, p) {
			fracture = append(fracture, p)
		}
		if testDefs.CanAffect(ConditionLaceration, p) {
			laceration = append(laceration, p)
		}
	}
	assert.Equal(t, []BodyPart{BodyPartArms, BodyPartHands, BodyPartLegs, BodyPartFeet}, fracture, "骨折は手足だけ")
	assert.Equal(t, []BodyPart{BodyPartHead, BodyPartTorso, BodyPartArms, BodyPartHands, BodyPartLegs, BodyPartFeet, BodyPartWholeBody}, laceration, "parts を持たない切り傷は全部位")
}

func TestConditionDefs_Def(t *testing.T) {
	t.Parallel()

	def, ok := testDefs.Def(ConditionLaceration)
	require.True(t, ok)
	assert.Equal(t, RecoverAfterTend, def.Recovery)
	assert.Equal(t, CauseBloodLoss, def.Cause)

	_, ok = testDefs.Def(ConditionType("unknown"))
	assert.False(t, ok, "未登録は ok=false")
}

func TestConditionDefs_HitWeight(t *testing.T) {
	t.Parallel()

	var got [BodyPartCount]float64
	for p := range BodyPartCount {
		got[p] = testDefs.HitWeight(p)
	}
	assert.Equal(t, [BodyPartCount]float64{2, 4, 3, 1, 3, 1, 0}, got)
}

func TestBodyPartByID(t *testing.T) {
	t.Parallel()

	for p := range BodyPartCount {
		got, ok := BodyPartByID(p.ID())
		require.True(t, ok, p.ID())
		assert.Equal(t, p, got)
	}
	_, ok := BodyPartByID("tail")
	assert.False(t, ok)
}

func TestConditionDefs_Injury(t *testing.T) {
	t.Parallel()

	ct, ok := testDefs.Injury(AttackFist)
	require.True(t, ok)
	assert.Equal(t, ConditionFracture, ct)

	_, ok = testDefs.Injury(AttackBow)
	assert.False(t, ok, "未登録は ok=false")
}
