package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemperatureSystem_String(t *testing.T) {
	t.Parallel()
	sys := &TemperatureSystem{}
	assert.Equal(t, "TemperatureSystem", sys.String())
}

func TestGetTileTemperatureAt(t *testing.T) {
	t.Parallel()

	t.Run("タイルが存在する場合は気温修正を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		tile := world.Manager.NewEntity()
		tile.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		tile.AddComponent(world.Components.TileTemperature, &gc.TileTemperature{
			Shelter: gc.ShelterFull,
		})

		result := getTileTemperatureAt(world, 5, 5)
		assert.Equal(t, 10, result)
	})

	t.Run("タイルが存在しない場合は0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		result := getTileTemperatureAt(world, 5, 5)
		assert.Equal(t, 0, result)
	})
}

func TestCalcTimerDelta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		effectiveTemp int
		expected      float64
	}{
		{"非常に寒い(0度以下)", -10, -0.5},
		{"非常に寒い(0度)", 0, -0.5},
		{"寒い(1-10度)", 5, -0.25},
		{"寒い(10度)", 10, -0.25},
		{"やや寒い(11-15度)", 12, 0},
		{"やや寒い(15度)", 15, 0},
		{"快適(16-25度)", 20, 0},
		{"快適(25度)", 25, 0},
		{"やや暑い(26-30度)", 28, 0},
		{"やや暑い(30度)", 30, 0},
		{"暑い(31-35度)", 33, 0.25},
		{"暑い(35度)", 35, 0.25},
		{"非常に暑い(36度以上)", 40, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := calcTimerDelta(tt.effectiveTemp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateTemperatureConditions(t *testing.T) {
	t.Parallel()

	t.Run("快適な温度では状態が回復する", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:  gc.ConditionHypothermia,
			Timer: 50,
		})

		updateTemperatureConditions(hs, 20, Insulation{}, false, 100, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		if cond != nil {
			assert.Less(t, cond.Timer, 50.0)
		}
	})

	t.Run("寒い環境で低体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}

		updateTemperatureConditions(hs, 0, Insulation{}, false, 100, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("暑い環境で高体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}

		updateTemperatureConditions(hs, 40, Insulation{}, false, 100, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHyperthermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("耐寒装備で低体温を軽減", func(t *testing.T) {
		t.Parallel()
		hs1 := &gc.HealthStatus{}
		hs2 := &gc.HealthStatus{}

		// 同じ寒い環境(0度)で比較
		updateTemperatureConditions(hs1, 0, Insulation{}, false, 100, 100)
		updateTemperatureConditions(hs2, 0, Insulation{Cold: 20}, false, 100, 100)

		cond1 := hs1.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		cond2 := hs2.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)

		require.NotNil(t, cond1)
		assert.Greater(t, cond1.Timer, 0.0)

		// 耐寒20で有効温度が20度になり快適範囲なので状態は追加されないか軽微
		if cond2 != nil {
			assert.Less(t, cond2.Timer, cond1.Timer)
		}
	})

	t.Run("Severity変化時にtrueを返す", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityNone,
			Timer:    24.5,
		})

		hasChange := updateTemperatureConditions(hs, 0, Insulation{}, false, 100, 100)
		assert.True(t, hasChange)
	})

	t.Run("Severity変化がない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}

		hasChange := updateTemperatureConditions(hs, 20, Insulation{}, false, 100, 100)
		assert.False(t, hasChange)
	})
}

func TestSeverityToMultiplier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity gc.Severity
		expected int
	}{
		{gc.SeverityNone, 0},
		{gc.SeverityMinor, 1},
		{gc.SeverityMedium, 2},
		{gc.SeveritySevere, 3},
	}

	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, severityToMultiplier(tt.severity))
		})
	}
}

func TestTemperatureSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("ダンジョンが設定されていない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = nil

		sys := &TemperatureSystem{}
		err := sys.Update(world)

		assert.Error(t, err)
	})

	t.Run("HealthStatusを持つエンティティの状態が更新される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon.DefinitionName = "亡者の森" // 基本気温0度

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "Ash")
		require.NoError(t, err)

		sys := &TemperatureSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		// 寒い環境なので低体温のタイマーが増加しているはず
		hs := world.Components.HealthStatus.Get(player).(*gc.HealthStatus)
		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("存在しないダンジョン名の場合はエラーなし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon.DefinitionName = "存在しないダンジョン"

		sys := &TemperatureSystem{}
		err := sys.Update(world)

		assert.NoError(t, err)
	})
}

func TestComfortableRange(t *testing.T) {
	t.Parallel()

	t.Run("断熱なしの場合の快適温度範囲", func(t *testing.T) {
		t.Parallel()
		lower, upper := ComfortableRange(Insulation{Cold: 0, Heat: 0})
		assert.Equal(t, 11, lower)
		assert.Equal(t, 30, upper)
	})

	t.Run("耐寒ありの場合は下限が下がる", func(t *testing.T) {
		t.Parallel()
		lower, upper := ComfortableRange(Insulation{Cold: 10, Heat: 0})
		assert.Equal(t, 1, lower)
		assert.Equal(t, 30, upper)
	})

	t.Run("耐熱ありの場合は上限が上がる", func(t *testing.T) {
		t.Parallel()
		lower, upper := ComfortableRange(Insulation{Cold: 0, Heat: 10})
		assert.Equal(t, 11, lower)
		assert.Equal(t, 40, upper)
	})

	t.Run("両方ありの場合", func(t *testing.T) {
		t.Parallel()
		lower, upper := ComfortableRange(Insulation{Cold: 15, Heat: 5})
		assert.Equal(t, -4, lower)
		assert.Equal(t, 35, upper)
	})
}

func TestCalculateEquippedInsulation(t *testing.T) {
	t.Parallel()

	t.Run("装備なしの場合は全て0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := worldhelper.SpawnPlayer(world, 0, 0, "Ash")
		require.NoError(t, err)

		insulation := CalculateEquippedInsulation(world, player)
		assert.Equal(t, 0, insulation.Cold)
		assert.Equal(t, 0, insulation.Heat)
	})

	t.Run("装備の断熱値が合算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := worldhelper.SpawnPlayer(world, 0, 0, "Ash")
		require.NoError(t, err)

		// 胴体装備（耐寒10, 耐熱5）
		armor := world.Manager.NewEntity()
		armor.AddComponent(world.Components.Wearable, &gc.Wearable{
			EquipmentCategory: gc.EquipmentTorso,
			InsulationCold:    10,
			InsulationHeat:    5,
		})
		armor.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{
			Owner: player,
		})

		// 頭装備（耐寒3, 耐熱2）
		helmet := world.Manager.NewEntity()
		helmet.AddComponent(world.Components.Wearable, &gc.Wearable{
			EquipmentCategory: gc.EquipmentHead,
			InsulationCold:    3,
			InsulationHeat:    2,
		})
		helmet.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{
			Owner: player,
		})

		insulation := CalculateEquippedInsulation(world, player)
		assert.Equal(t, 13, insulation.Cold)
		assert.Equal(t, 7, insulation.Heat)
	})
}

func TestCalculateHypothermiaEffects(t *testing.T) {
	t.Parallel()

	t.Run("全身にSTR,VIT,DEX,AGIペナルティを与える", func(t *testing.T) {
		t.Parallel()
		effects := calculateHypothermiaEffects(gc.SeverityMinor)
		require.Len(t, effects, 4)
		assert.Equal(t, gc.StatStrength, effects[0].Stat)
		assert.Equal(t, gc.StatVitality, effects[1].Stat)
		assert.Equal(t, gc.StatDexterity, effects[2].Stat)
		assert.Equal(t, gc.StatAgility, effects[3].Stat)
		for _, e := range effects {
			assert.Equal(t, -1, e.Value)
		}
	})

	t.Run("SeverityNoneでは効果なし", func(t *testing.T) {
		t.Parallel()
		effects := calculateHypothermiaEffects(gc.SeverityNone)
		assert.Nil(t, effects)
	})

	t.Run("重度の方が効果が大きい", func(t *testing.T) {
		t.Parallel()
		minor := calculateHypothermiaEffects(gc.SeverityMinor)
		severe := calculateHypothermiaEffects(gc.SeveritySevere)
		assert.Greater(t, -severe[0].Value, -minor[0].Value)
	})
}

func TestCalculateHyperthermiaEffects(t *testing.T) {
	t.Parallel()

	t.Run("全身にSTR,SEN,VITペナルティを与える", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.SeverityMinor)
		require.Len(t, effects, 3)
		assert.Equal(t, gc.StatStrength, effects[0].Stat)
		assert.Equal(t, gc.StatSensation, effects[1].Stat)
		assert.Equal(t, gc.StatVitality, effects[2].Stat)
		for _, e := range effects {
			assert.Equal(t, -1, e.Value)
		}
	})

	t.Run("SeverityNoneでは効果なし", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.SeverityNone)
		assert.Nil(t, effects)
	})

	t.Run("重度の方が効果が大きい", func(t *testing.T) {
		t.Parallel()
		minor := calculateHyperthermiaEffects(gc.SeverityMinor)
		severe := calculateHyperthermiaEffects(gc.SeveritySevere)
		assert.Equal(t, -1, minor[0].Value)
		assert.Equal(t, -3, severe[0].Value)
	})
}

func TestUpdateConditionEffects(t *testing.T) {
	t.Parallel()

	t.Run("低体温の効果が適用される", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityMinor,
			Timer:    30,
		})

		updateConditionEffects(partHealth)

		cond := partHealth.GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		require.Len(t, cond.Effects, 4) // STR, VIT, DEX, AGI
	})

	t.Run("高体温の効果が適用される", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHyperthermia,
			Severity: gc.SeverityMedium,
			Timer:    50,
		})

		updateConditionEffects(partHealth)

		cond := partHealth.GetCondition(gc.ConditionHyperthermia)
		require.NotNil(t, cond)
		require.Len(t, cond.Effects, 3) // STR, SEN, VIT
	})
}

func TestLogTemperatureChange(t *testing.T) {
	t.Parallel()

	t.Run("悪化時のメッセージが取得できる", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeverityMinor)
		assert.Contains(t, msg, "冷えてきた")
	})

	t.Run("中程度悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeverityMedium)
		assert.Contains(t, msg, "かなり冷えている")
	})

	t.Run("重度悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeveritySevere)
		assert.Contains(t, msg, "危険な状態")
	})

	t.Run("高体温悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHyperthermia, gc.SeverityMinor)
		assert.Contains(t, msg, "火照ってきた")
	})

	t.Run("回復時のメッセージが取得できる", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.ConditionHypothermia, gc.SeverityNone)
		assert.Contains(t, msg, "温まった")
	})

	t.Run("部分回復のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.ConditionHypothermia, gc.SeverityMinor)
		assert.Contains(t, msg, "少し温まってきた")
	})

	t.Run("高体温回復のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.ConditionHyperthermia, gc.SeverityNone)
		assert.Contains(t, msg, "涼しくなった")
	})

	t.Run("SeverityNoneの悪化メッセージは空", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeverityNone)
		assert.Empty(t, msg)
	})

	t.Run("SeveritySevereの回復メッセージは空", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.ConditionHypothermia, gc.SeveritySevere)
		assert.Empty(t, msg)
	})
}
