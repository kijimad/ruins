package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDefs は components のテストが使う不調の定義の表。値は raw の行と同じにする
var testDefs = NewConditionDefs(map[ConditionType]ConditionDef{
	ConditionType("hypothermia"): {
		Name:                    "Hypothermia",
		PainPerSeverity:         6,
		BodyFuncDropPerSeverity: 10,
		BloodDropPerSeverity:    22,
		Cause:                   CauseFrozen,
		Onset:                   OnsetCold,
	},
	ConditionType("fracture"): {
		Name:                    "Fracture",
		PainPerSeverity:         18,
		BodyFuncDropPerSeverity: 20,
		Recovery:                RecoverAfterTend,
		RecoverPer:              1,
		Parts:                   []BodyPart{BodyPartArms, BodyPartHands, BodyPartLegs, BodyPartFeet},
	},
	ConditionType("laceration"): {
		Name:                    "Laceration",
		PainPerSeverity:         8,
		BodyFuncDropPerSeverity: 8,
		Recovery:                RecoverAfterTend,
		RecoverPer:              2,
		BloodDropPerSeverity:    25,
		Cause:                   CauseBloodLoss,
	},
	ConditionType("liver_illness"): {
		Name:                    "Liver illness",
		PainPerSeverity:         4,
		BodyFuncDropPerSeverity: 10,
		Recovery:                ProgressUntilTend,
		WorsenPer:               2,
		RecoverPer:              1,
		BloodDropPerSeverity:    25,
		Cause:                   CauseIllness,
	},
	ConditionType("food_poisoning"): {
		Name:                    "Food poisoning",
		PainPerSeverity:         5,
		BodyFuncDropPerSeverity: 12,
		Recovery:                RecoverOverTime,
		RecoverPer:              2,
		Onset:                   OnsetRottenFood,
	},
	ConditionType("exhaustion"): {
		Name:                    "Exhaustion",
		BodyFuncDropPerSeverity: 10,
		Onset:                   OnsetFatigue,
	},
	ConditionType("malnutrition"): {
		Name:                    "Malnutrition",
		BodyFuncDropPerSeverity: 10,
		Onset:                   OnsetHunger,
	},
}, [BodyPartCount]BodyPartDef{
	BodyPartHead:      {Name: "Head", HitWeight: 2},
	BodyPartTorso:     {Name: "Torso", HitWeight: 4},
	BodyPartArms:      {Name: "Arm", HitWeight: 3},
	BodyPartHands:     {Name: "Hand", HitWeight: 1},
	BodyPartLegs:      {Name: "Leg", HitWeight: 3},
	BodyPartFeet:      {Name: "Foot", HitWeight: 1},
	BodyPartWholeBody: {Name: "Whole body"},
}, map[string]ConditionType{
	AttackFist.Type:  ConditionType("fracture"),
	AttackSword.Type: ConditionType("laceration"),
})

func TestConditionDefs_CanAffect(t *testing.T) {
	t.Parallel()

	var fracture []BodyPart
	var laceration []BodyPart
	var unknown []BodyPart
	for p := range BodyPartCount {
		if testDefs.CanAffect(ConditionType("fracture"), p) {
			fracture = append(fracture, p)
		}
		if testDefs.CanAffect(ConditionType("laceration"), p) {
			laceration = append(laceration, p)
		}
		if testDefs.CanAffect(ConditionType("unknown"), p) {
			unknown = append(unknown, p)
		}
	}
	assert.Equal(t, []BodyPart{BodyPartArms, BodyPartHands, BodyPartLegs, BodyPartFeet}, fracture, "骨折は手足だけ")
	assert.Equal(t, []BodyPart{BodyPartHead, BodyPartTorso, BodyPartArms, BodyPartHands, BodyPartLegs, BodyPartFeet, BodyPartWholeBody}, laceration, "parts を持たない切り傷は全部位")
	assert.Equal(t, []BodyPart(nil), unknown, "未登録の種類はどこにも起こらない")
}

func TestConditionDefs_Def(t *testing.T) {
	t.Parallel()

	def, ok := testDefs.Def(ConditionType("laceration"))
	require.True(t, ok)
	assert.Equal(t, RecoverAfterTend, def.Recovery)
	assert.Equal(t, CauseBloodLoss, def.Cause)

	_, ok = testDefs.Def(ConditionType("unknown"))
	assert.False(t, ok, "未登録は ok=false")
}

func TestConditionDefs_未登録の種類は痛みも機能低下も出血も無い(t *testing.T) {
	t.Parallel()

	cond := &HealthCondition{Type: ConditionType("unknown"), Severity: SeveritySevere}
	pain, _, drop := testDefs.BodyFuncImpact(cond, BodyPartTorso)
	assert.Equal(t, [3]int{0, 0, 0}, [3]int{pain, drop, testDefs.BloodDrop(cond)})
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
	assert.Equal(t, ConditionType("fracture"), ct)

	_, ok = testDefs.Injury(AttackBow)
	assert.False(t, ok, "未登録は ok=false")
}

func TestConditionDefs_ByOnset(t *testing.T) {
	t.Parallel()

	got := map[ConditionOnset]ConditionType{}
	for _, o := range []ConditionOnset{OnsetCold, OnsetRottenFood, OnsetFatigue, OnsetHunger} {
		got[o] = testDefs.ByOnset(o)
	}
	assert.Equal(t, map[ConditionOnset]ConditionType{
		OnsetCold:       "hypothermia",
		OnsetRottenFood: "food_poisoning",
		OnsetFatigue:    "exhaustion",
		OnsetHunger:     "malnutrition",
	}, got)
	assert.Panics(t, func() { ConditionDefs{}.ByOnset(OnsetCold) })
}

func TestConditionDefs_Types(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []ConditionType{
		"exhaustion", "food_poisoning", "fracture", "hypothermia", "laceration", "liver_illness", "malnutrition",
	}, testDefs.Types())
}
