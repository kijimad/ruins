package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCondition_DisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cond     HealthCondition
		expected string
	}{
		{
			name:     "低体温（軽）",
			cond:     HealthCondition{Type: ConditionHypothermia, Severity: SeverityMinor},
			expected: "低体温(軽)",
		},
		{
			name:     "高体温（重）",
			cond:     HealthCondition{Type: ConditionHyperthermia, Severity: SeveritySevere},
			expected: "高体温(重)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.cond.DisplayName())
		})
	}
}

func TestBodyPartHealth_SetCondition(t *testing.T) {
	t.Parallel()

	bph := &BodyPartHealth{}

	// 状態を追加
	bph.SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMinor,
		Effects:  []StatEffect{{Stat: StatStrength, Value: -1}},
	})

	assert.Equal(t, 1, len(bph.Conditions))
	assert.Equal(t, SeverityMinor, bph.Conditions[0].Severity)

	// 同じ状態を上書き
	bph.SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeveritySevere,
		Effects:  []StatEffect{{Stat: StatStrength, Value: -3}},
	})

	assert.Equal(t, 1, len(bph.Conditions))
	assert.Equal(t, SeveritySevere, bph.Conditions[0].Severity)
}

func TestBodyPartHealth_RemoveCondition(t *testing.T) {
	t.Parallel()

	bph := &BodyPartHealth{}

	bph.SetCondition(HealthCondition{Type: ConditionHypothermia, Severity: SeverityMinor})
	bph.SetCondition(HealthCondition{Type: ConditionHyperthermia, Severity: SeveritySevere})

	assert.Equal(t, 2, len(bph.Conditions))

	bph.RemoveCondition(ConditionHypothermia)

	assert.Equal(t, 1, len(bph.Conditions))
	assert.Equal(t, ConditionHyperthermia, bph.Conditions[0].Type)
}

func TestHealthStatus_GetStatModifier(t *testing.T) {
	t.Parallel()

	hs := &HealthStatus{}

	// 複数部位から同じステータスに影響
	hs.Parts[BodyPartTorso].SetCondition(HealthCondition{
		Type:    ConditionHypothermia,
		Effects: []StatEffect{{Stat: StatStrength, Value: -2}},
	})
	hs.Parts[BodyPartArms].SetCondition(HealthCondition{
		Type:    ConditionHypothermia,
		Effects: []StatEffect{{Stat: StatStrength, Value: -1}},
	})

	// 合計 -3 になるはず
	assert.Equal(t, -3, hs.GetStatModifier(StatStrength))

	// 影響のないステータスは 0
	assert.Equal(t, 0, hs.GetStatModifier(StatAgility))
}
