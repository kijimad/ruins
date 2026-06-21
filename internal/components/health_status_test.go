package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeverity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{"None", SeverityNone, ""},
		{"Minor", SeverityMinor, "軽"},
		{"Medium", SeverityMedium, "中"},
		{"Severe", SeveritySevere, "重"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.severity.String())
		})
	}
}

func TestSeverity_String_Panic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = Severity(99).String()
	})
}

func TestStatType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		statType StatType
		want     string
	}{
		{StatVitality, "体力"},
		{StatStrength, "筋力"},
		{StatSensation, "感覚"},
		{StatDexterity, "器用"},
		{StatAgility, "敏捷"},
		{StatDefense, "防御"},
		{StatType("Unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.statType), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.statType.String())
		})
	}
}

func TestTimerToSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		timer float64
		want  Severity
	}{
		{"0はNone", 0, SeverityNone},
		{"24.9はNone", 24.9, SeverityNone},
		{"25はMinor", 25, SeverityMinor},
		{"49.9はMinor", 49.9, SeverityMinor},
		{"50はMedium", 50, SeverityMedium},
		{"74.9はMedium", 74.9, SeverityMedium},
		{"75はSevere", 75, SeveritySevere},
		{"100はSevere", 100, SeveritySevere},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TimerToSeverity(tt.timer))
		})
	}
}

func TestHealthCondition_UpdateTimer(t *testing.T) {
	t.Parallel()

	t.Run("悪化してSeverityが変わる", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Timer: 20}
		prev, current := hc.UpdateTimer(10)
		assert.Equal(t, SeverityNone, prev)
		assert.Equal(t, SeverityMinor, current)
		assert.InDelta(t, 30.0, hc.Timer, 0.001)
	})

	t.Run("回復してSeverityが変わる", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Timer: 30, Severity: SeverityMinor}
		prev, current := hc.UpdateTimer(-10)
		assert.Equal(t, SeverityMinor, prev)
		assert.Equal(t, SeverityNone, current)
	})

	t.Run("タイマーは0未満にならない", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Timer: 5}
		hc.UpdateTimer(-20)
		assert.InDelta(t, 0.0, hc.Timer, 0.001)
	})

	t.Run("タイマーは100を超えない", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Timer: 95}
		hc.UpdateTimer(20)
		assert.InDelta(t, 100.0, hc.Timer, 0.001)
	})
}

func TestHealthCondition_IsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		timer  float64
		active bool
	}{
		{"Timer=0は非アクティブ", 0, false},
		{"Timer=24.9は非アクティブ", 24.9, false},
		{"Timer=25はアクティブ", 25, true},
		{"Timer=100はアクティブ", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hc := &HealthCondition{Timer: tt.timer}
			assert.Equal(t, tt.active, hc.IsActive())
		})
	}
}

func TestHealthCondition_DisplayName(t *testing.T) {
	t.Parallel()

	t.Run("SeverityNoneは重症度表示なし", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Severity: SeverityNone}
		assert.Equal(t, "低体温", hc.DisplayName())
	})

	t.Run("低体温で軽度", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Severity: SeverityMinor}
		assert.Equal(t, "低体温(軽)", hc.DisplayName())
	})

	t.Run("高体温で重度", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHyperthermia, Severity: SeveritySevere}
		assert.Equal(t, "高体温(重)", hc.DisplayName())
	})
}

func TestConditionTypeDisplayName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "低体温", ConditionTypeDisplayName(ConditionHypothermia))
	assert.Equal(t, "高体温", ConditionTypeDisplayName(ConditionHyperthermia))
	assert.Equal(t, "Unknown", ConditionTypeDisplayName(ConditionType("Unknown")))
}

func TestBodyPartHealth_SetCondition(t *testing.T) {
	t.Parallel()

	t.Run("新規追加", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{}
		bph.SetCondition(HealthCondition{Type: ConditionHypothermia, Timer: 30})
		require.Len(t, bph.Conditions, 1)
		assert.InDelta(t, 30.0, bph.Conditions[0].Timer, 0.001)
	})

	t.Run("同種の状態は上書き", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{}
		bph.SetCondition(HealthCondition{Type: ConditionHypothermia, Severity: SeverityMinor})
		bph.SetCondition(HealthCondition{Type: ConditionHypothermia, Severity: SeveritySevere})
		require.Len(t, bph.Conditions, 1)
		assert.Equal(t, SeveritySevere, bph.Conditions[0].Severity)
	})
}

func TestBodyPartHealth_RemoveCondition(t *testing.T) {
	t.Parallel()

	t.Run("存在する状態を削除", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{
			Conditions: []HealthCondition{
				{Type: ConditionHypothermia},
				{Type: ConditionHyperthermia},
			},
		}
		bph.RemoveCondition(ConditionHypothermia)
		require.Len(t, bph.Conditions, 1)
		assert.Equal(t, ConditionHyperthermia, bph.Conditions[0].Type)
	})

	t.Run("存在しない状態の削除は何もしない", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{}
		bph.RemoveCondition(ConditionHypothermia)
		assert.Empty(t, bph.Conditions)
	})
}

func TestBodyPartHealth_GetCondition(t *testing.T) {
	t.Parallel()

	bph := &BodyPartHealth{
		Conditions: []HealthCondition{
			{Type: ConditionHypothermia, Timer: 50},
		},
	}

	cond := bph.GetCondition(ConditionHypothermia)
	require.NotNil(t, cond)
	assert.InDelta(t, 50.0, cond.Timer, 0.001)

	assert.Nil(t, bph.GetCondition(ConditionHyperthermia))
}

func TestBodyPartHealth_GetOrCreateCondition(t *testing.T) {
	t.Parallel()

	t.Run("既存を取得", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{
			Conditions: []HealthCondition{
				{Type: ConditionHypothermia, Timer: 50},
			},
		}
		cond := bph.GetOrCreateCondition(ConditionHypothermia)
		assert.InDelta(t, 50.0, cond.Timer, 0.001)
		assert.Len(t, bph.Conditions, 1)
	})

	t.Run("新規作成", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{}
		cond := bph.GetOrCreateCondition(ConditionHypothermia)
		assert.InDelta(t, 0.0, cond.Timer, 0.001)
		assert.Equal(t, SeverityNone, cond.Severity)
		assert.Len(t, bph.Conditions, 1)
	})
}

func TestBodyPartHealth_UpdateConditionTimer(t *testing.T) {
	t.Parallel()

	t.Run("タイマー更新でSeverity変化を返す", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{}
		change := bph.UpdateConditionTimer(ConditionHypothermia, 30)
		assert.Equal(t, SeverityNone, change.Prev)
		assert.Equal(t, SeverityMinor, change.Current)
		assert.Equal(t, ConditionHypothermia, change.CondType)
	})

	t.Run("タイマーが0になったら状態を削除する", func(t *testing.T) {
		t.Parallel()
		bph := &BodyPartHealth{
			Conditions: []HealthCondition{
				{Type: ConditionHypothermia, Timer: 5},
			},
		}
		bph.UpdateConditionTimer(ConditionHypothermia, -10)
		assert.Empty(t, bph.Conditions)
	})
}

func TestHealthStatus_GetStatModifier(t *testing.T) {
	t.Parallel()

	t.Run("修正値なし", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		assert.Equal(t, 0, hs.GetStatModifier(StatStrength))
	})

	t.Run("複数部位の修正値を合算", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartTorso].SetCondition(HealthCondition{
			Type:    ConditionHypothermia,
			Effects: []StatEffect{{Stat: StatStrength, Value: -2}},
		})
		hs.Parts[BodyPartArms].SetCondition(HealthCondition{
			Type:    ConditionHypothermia,
			Effects: []StatEffect{{Stat: StatStrength, Value: -1}, {Stat: StatAgility, Value: -1}},
		})
		assert.Equal(t, -3, hs.GetStatModifier(StatStrength))
		assert.Equal(t, -1, hs.GetStatModifier(StatAgility))
		assert.Equal(t, 0, hs.GetStatModifier(StatDefense))
	})
}

func TestClamp(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 5, clamp(5, 0, 10))
		assert.Equal(t, 0, clamp(-1, 0, 10))
		assert.Equal(t, 10, clamp(15, 0, 10))
	})

	t.Run("float64", func(t *testing.T) {
		t.Parallel()
		assert.InDelta(t, 5.0, clamp(5.0, 0.0, 10.0), 0.001)
		assert.InDelta(t, 0.0, clamp(-1.0, 0.0, 10.0), 0.001)
		assert.InDelta(t, 10.0, clamp(15.0, 0.0, 10.0), 0.001)
	})
}
