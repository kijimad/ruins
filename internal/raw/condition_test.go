package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeConditionRaws は Go の不調と部位すべてに行を持つ検証を通る raws を作り、mutate で崩す
func makeConditionRaws(mutate func(*oapi.Raws)) oapi.Raws {
	bodyParts := make([]oapi.BodyPart, 0, gc.BodyPartCount)
	for bp := range gc.BodyPartCount {
		bodyParts = append(bodyParts, oapi.BodyPart{Id: bp.ID(), Name: bp.String(), HitWeight: 1})
	}
	conditions := make([]oapi.Condition, 0, len(gc.ConditionTypes()))
	for _, ct := range gc.ConditionTypes() {
		conditions = append(conditions, oapi.Condition{Id: string(ct), Name: string(ct), Description: string(ct)})
	}
	raws := oapi.Raws{
		BodyParts:  &bodyParts,
		Conditions: &conditions,
		Items: &[]oapi.Item{{
			Id:     "splint",
			Remedy: &oapi.Remedy{Treats: []oapi.ConditionTypeName{string(gc.ConditionFracture)}},
		}},
	}
	mutate(&raws)
	return raws
}

func TestValidateConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "全行がそろい参照が実在すれば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "部位を限った不調は命中しうる部位があれば通る",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"arm", "leg"}
			},
		},
		{
			name: "部位の id 重複",
			mutate: func(r *oapi.Raws) {
				*r.BodyParts = append(*r.BodyParts, oapi.BodyPart{Id: "arm", Name: "Arm"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "不調の id 重複",
			mutate: func(r *oapi.Raws) {
				*r.Conditions = append(*r.Conditions, oapi.Condition{Id: "fracture"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "Go に無い部位の行",
			mutate: func(r *oapi.Raws) {
				*r.BodyParts = append(*r.BodyParts, oapi.BodyPart{Id: "tail", Name: "Tail"})
			},
			wantErr: errBodyPartUnknown,
		},
		{
			name: "不調の部位が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"tail"}
			},
			wantErr: errConditionPartUndefined,
		},
		{
			name: "不調の部位に全身を含む",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"arm", "whole_body"}
			},
			wantErr: errConditionPartWholeBody,
		},
		{
			name: "不調の部位がどれも命中しない",
			mutate: func(r *oapi.Raws) {
				(*r.BodyParts)[gc.BodyPartArms].HitWeight = 0
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"arm"}
			},
			wantErr: errConditionNoHitPart,
		},
		{
			name: "全部位の不調でも命中する部位が無い",
			mutate: func(r *oapi.Raws) {
				for i := range *r.BodyParts {
					(*r.BodyParts)[i].HitWeight = 0
				}
			},
			wantErr: errConditionNoHitPart,
		},
		{
			name: "治療の対象が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.Items)[0].Remedy.Treats = []oapi.ConditionTypeName{"Fracture"}
			},
			wantErr: errRemedyConditionUndefined,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateConditions(makeConditionRaws(tt.mutate))
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateCodeRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "Go の不調と部位すべてに行があれば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "部位の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.BodyParts = (*r.BodyParts)[:gc.BodyPartWholeBody]
			},
			wantErr: errBodyPartRowMissing,
		},
		{
			name: "不調の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.Conditions = (*r.Conditions)[1:]
			},
			wantErr: errConditionRowMissing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCodeRows(makeConditionRaws(tt.mutate))
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewConditionDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs, err := NewConditionDefs(raws)
	require.NoError(t, err)

	tests := []struct {
		ct   gc.ConditionType
		want gc.ConditionDef
	}{
		{gc.ConditionHypothermia, gc.ConditionDef{
			Name: "Hypothermia", Description: "The body is dangerously cold. Warm up to recover.",
			PainPerSeverity: 6, BodyFuncDropPerSeverity: 10, BloodDropPerSeverity: 22, Cause: gc.CauseFrozen,
		}},
		{gc.ConditionFracture, gc.ConditionDef{
			Name: "Fracture", Description: "A broken bone. It will not heal until treated.",
			PainPerSeverity: 18, BodyFuncDropPerSeverity: 20, Recovery: gc.RecoverAfterTend, RecoverPer: 1,
			Parts: []gc.BodyPart{gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet},
		}},
		{gc.ConditionLaceration, gc.ConditionDef{
			Name: "Laceration", Description: "An open wound. It will not heal until treated.",
			PainPerSeverity: 8, BodyFuncDropPerSeverity: 8, Recovery: gc.RecoverAfterTend, RecoverPer: 2,
			BloodDropPerSeverity: 25, Cause: gc.CauseBloodLoss,
		}},
		{gc.ConditionLiverIllness, gc.ConditionDef{
			Name: "Liver illness", Description: "It worsens while untreated and drains HP when severe.",
			PainPerSeverity: 4, BodyFuncDropPerSeverity: 10, Recovery: gc.ProgressUntilTend, WorsenPer: 2, RecoverPer: 1,
			BloodDropPerSeverity: 25, Cause: gc.CauseIllness,
		}},
		{gc.ConditionFoodPoisoning, gc.ConditionDef{
			Name: "Food poisoning", Description: "Nausea from bad food. It clears on its own over time.",
			PainPerSeverity: 5, BodyFuncDropPerSeverity: 12, Recovery: gc.RecoverOverTime, RecoverPer: 2,
		}},
		{gc.ConditionExhaustion, gc.ConditionDef{
			Name: "Exhaustion", Description: "Worn out from lack of rest. Sleep to recover.",
			BodyFuncDropPerSeverity: 10,
		}},
		{gc.ConditionMalnutrition, gc.ConditionDef{
			Name: "Malnutrition", Description: "Weakened by hunger. Eat to recover.",
			BodyFuncDropPerSeverity: 10,
		}},
	}
	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			t.Parallel()
			got, ok := defs.Def(tt.ct)
			require.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}

	var weights [gc.BodyPartCount]float64
	for bp := range gc.BodyPartCount {
		weights[bp] = defs.HitWeight(bp)
	}
	assert.Equal(t, [gc.BodyPartCount]float64{2, 4, 3, 1, 3, 1, 0}, weights)
}
