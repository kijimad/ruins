package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeverity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s    Severity
		want string
	}{
		{SeverityNone, ""},
		{SeverityMinor, "Minor"},
		{SeverityMedium, "Medium"},
		{SeveritySevere, "Severe"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.s.String())
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
		st   StatType
		want string
	}{
		{StatVitality, "Vitality"},
		{StatStrength, "Strength"},
		{StatSensation, "Sensation"},
		{StatDexterity, "Dexterity"},
		{StatAgility, "Agility"},
		{StatDefense, "Defense"},
		{StatType("Unknown"), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.st.String())
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
		assert.Equal(t, "Hypothermia", hc.DisplayName())
	})

	t.Run("低体温で軽度", func(t *testing.T) {
		t.Parallel()
		hc := &HealthCondition{Type: ConditionHypothermia, Severity: SeverityMinor}
		assert.Equal(t, "Hypothermia Minor", hc.DisplayName())
	})
}

func TestConditionTypeDisplayName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Hypothermia", ConditionTypeDisplayName(ConditionHypothermia))
	assert.Equal(t, "Unknown", ConditionTypeDisplayName(ConditionType("Unknown")))
}

func TestConditionTypeDescription(t *testing.T) {
	t.Parallel()

	assert.Contains(t, ConditionTypeDescription(ConditionFracture), "broken bone")
	assert.Empty(t, ConditionTypeDescription(ConditionType("Unknown")), "未登録は空文字")
}

func TestConditionDefFor(t *testing.T) {
	t.Parallel()

	// 怪我・病気は Recovery を持つ。低体温は表に載るが Recovery を持たず TemperatureSystem 管轄
	def, ok := ConditionDefFor(ConditionLaceration)
	require.True(t, ok)
	assert.Equal(t, RecoverAfterTend, def.Recovery)
	assert.Equal(t, 1, def.BleedPer, "切り傷は失血する")

	cold, ok := ConditionDefFor(ConditionHypothermia)
	require.True(t, ok, "低体温も表示のため表には載る")
	assert.Empty(t, cold.Recovery, "低体温は Recovery を持たない")

	_, ok = ConditionDefFor(ConditionType("Unknown"))
	assert.False(t, ok, "未登録は ok=false")
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
				{Type: ConditionType("test")},
			},
		}
		bph.RemoveCondition(ConditionHypothermia)
		require.Len(t, bph.Conditions, 1)
		assert.Equal(t, ConditionType("test"), bph.Conditions[0].Type)
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

	assert.Nil(t, bph.GetCondition(ConditionType("test")))
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

func TestHealthStatus_Capacities(t *testing.T) {
	t.Parallel()

	t.Run("不調なしは全機能100で痛み0", func(t *testing.T) {
		t.Parallel()
		caps := (&HealthStatus{}).Capacities()
		assert.Equal(t, BodyCapacities{Pain: 0, Consciousness: 100, Manipulation: 100, Moving: 100, Sight: 100}, caps)
	})

	t.Run("腕の骨折は操作を下げ痛みを与え意識を落とす", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionFracture, Severity: SeverityMedium})
		caps := hs.Capacities()
		// 骨折 18/20 の中度。痛み=18*2=36、意識=100-36/2=82、操作=(100-20*2)*82/100=49、
		// 歩行と視覚は局所低下なしだが意識が掛かって82
		assert.Equal(t, BodyCapacities{Pain: 36, Consciousness: 82, Manipulation: 49, Moving: 82, Sight: 82}, caps)
	})

	t.Run("部位で下げる機能が変わる", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartLegs].SetCondition(HealthCondition{Type: ConditionFracture, Severity: SeverityMinor})
		caps := hs.Capacities()
		assert.Less(t, int(caps.Moving), int(caps.Sight), "脚の怪我は歩行を下げ視覚は下げない")
	})
}

func TestBodyPartMetas_全部位が登録されている(t *testing.T) {
	t.Parallel()

	// 配列テーブルは行を忘れてもゼロ値で埋まり switch の網羅チェックが効かないので、
	// 全部位に表示名と機能が入っていることをテストで担保する
	for bp := range BodyPartCount {
		assert.NotEmpty(t, bodyPartMetas[bp].displayName, "部位 %d に表示名がある", bp)
		assert.NotEmpty(t, bodyPartMetas[bp].capacity, "部位 %d に身体機能がある", bp)
	}
}

func TestHealthStatus_IsBleeding(t *testing.T) {
	t.Parallel()

	t.Run("未治療で発症中の切り傷は出血中", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionLaceration, Timer: 60, Severity: TimerToSeverity(60)})
		assert.True(t, hs.IsBleeding())
	})

	t.Run("治療した切り傷は出血しない", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionLaceration, Timer: 60, Severity: TimerToSeverity(60), TendQuality: 100})
		assert.False(t, hs.IsBleeding())
	})

	t.Run("発症前の掠り傷は出血しない", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionLaceration, Timer: 10, Severity: TimerToSeverity(10)})
		assert.False(t, hs.IsBleeding())
	})

	t.Run("骨折は出血しない", func(t *testing.T) {
		t.Parallel()
		hs := &HealthStatus{}
		hs.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionFracture, Timer: 60, Severity: TimerToSeverity(60)})
		assert.False(t, hs.IsBleeding())
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

func TestConditionCapacityImpact(t *testing.T) {
	t.Parallel()

	t.Run("腕の骨折は操作を下げ痛みを与える", func(t *testing.T) {
		t.Parallel()
		// 骨折 18/20 の中度: 痛み 18*2=36、操作 20*2=40
		pain, capacity, drop := ConditionCapacityImpact(ConditionFracture, BodyPartArms, SeverityMedium)
		assert.Equal(t, 36, pain)
		assert.Equal(t, CapacityManipulation, capacity)
		assert.Equal(t, 40, drop)
	})

	t.Run("症状ごとに反応率が違う", func(t *testing.T) {
		t.Parallel()
		// 同じ部位・重症度でも切り傷は骨折より痛みも機能低下も小さい
		fracPain, _, fracDrop := ConditionCapacityImpact(ConditionFracture, BodyPartArms, SeverityMedium)
		lacPain, _, lacDrop := ConditionCapacityImpact(ConditionLaceration, BodyPartArms, SeverityMedium)
		assert.Less(t, lacPain, fracPain, "切り傷は骨折より痛みが小さい")
		assert.Less(t, lacDrop, fracDrop, "切り傷は骨折より機能低下が小さい")
	})

	t.Run("脚の不調は歩行を下げる", func(t *testing.T) {
		t.Parallel()
		_, capacity, _ := ConditionCapacityImpact(ConditionFracture, BodyPartFeet, SeverityMinor)
		assert.Equal(t, CapacityMoving, capacity)
	})

	t.Run("重症度なしは影響なし", func(t *testing.T) {
		t.Parallel()
		// capacity は部位で定まり重症度に依らない。影響なしは drop と pain が0であることで表す
		pain, capacity, drop := ConditionCapacityImpact(ConditionFracture, BodyPartArms, SeverityNone)
		assert.Equal(t, 0, pain)
		assert.Equal(t, CapacityManipulation, capacity)
		assert.Equal(t, 0, drop)
	})
}
