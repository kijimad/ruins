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

		// タイルエンティティを作成
		tile := world.Manager.NewEntity()
		tile.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		tile.AddComponent(world.Components.TileTemperature, &gc.TileTemperature{
			Shelter: gc.ShelterFull,
		})

		result := getTileTemperatureAt(world, 5, 5)
		assert.Equal(t, 10, result) // ShelterFull = 10
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
		// 事前に低体温状態を設定
		hs.Parts[gc.BodyPartTorso].SetCondition(gc.HealthCondition{
			Type:  gc.ConditionHypothermia,
			Timer: 50,
		})

		insulation := [gc.BodyPartCount]Insulation{}
		updateTemperatureConditions(hs, 20, insulation, false)

		// タイマーが減少しているはず
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
		if cond != nil {
			assert.Less(t, cond.Timer, 50.0)
		}
	})

	t.Run("寒い環境で低体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		insulation := [gc.BodyPartCount]Insulation{}

		// 非常に寒い環境
		updateTemperatureConditions(hs, 0, insulation, false)

		// 低体温の状態が追加されているはず
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("暑い環境で高体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		insulation := [gc.BodyPartCount]Insulation{}

		// 非常に暑い環境
		updateTemperatureConditions(hs, 40, insulation, false)

		// 高体温の状態が追加されているはず
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHyperthermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("耐寒装備で低体温を軽減", func(t *testing.T) {
		t.Parallel()
		hs1 := &gc.HealthStatus{}
		hs2 := &gc.HealthStatus{}
		noInsulation := [gc.BodyPartCount]Insulation{}
		withInsulation := [gc.BodyPartCount]Insulation{}
		withInsulation[gc.BodyPartTorso] = Insulation{Cold: 20} // 耐寒+20

		// 同じ寒い環境(0度)で比較
		updateTemperatureConditions(hs1, 0, noInsulation, false)
		updateTemperatureConditions(hs2, 0, withInsulation, false)

		// 耐寒ありの方がタイマー増加が少ないはず
		cond1 := hs1.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
		cond2 := hs2.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)

		// 耐寒なしは状態が追加される
		require.NotNil(t, cond1)
		assert.Greater(t, cond1.Timer, 0.0)

		// 耐寒20で有効温度が20度になり、快適範囲なので状態は追加されないか軽微
		if cond2 != nil {
			assert.Less(t, cond2.Timer, cond1.Timer)
		}
	})

	t.Run("Severity変化時にtrueを返す", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		// タイマーを閾値付近に設定（24.5 → 25で SeverityNone → SeverityMinor に変化）
		hs.Parts[gc.BodyPartTorso].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityNone,
			Timer:    24.5,
		})
		insulation := [gc.BodyPartCount]Insulation{}

		// 非常に寒い環境（delta=+0.5）でSeverityがNone→Minorに変化
		hasChange := updateTemperatureConditions(hs, 0, insulation, false)
		assert.True(t, hasChange)
	})

	t.Run("Severity変化がない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		insulation := [gc.BodyPartCount]Insulation{}

		// 快適な温度で初期状態ならSeverity変化なし
		hasChange := updateTemperatureConditions(hs, 20, insulation, false)
		assert.False(t, hasChange)
	})
}

func TestUpdateFrostbiteTimer(t *testing.T) {
	t.Parallel()

	t.Run("非常に寒い環境で凍傷タイマーが増加", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}

		updateFrostbiteTimer(partHealth, 0)

		cond := partHealth.GetCondition(gc.ConditionFrostbite)
		require.NotNil(t, cond)
		assert.Equal(t, 0.5, cond.Timer)
	})

	t.Run("危険な環境で凍傷タイマーが増加", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}

		// 有効温度1-5度は危険レベル
		updateFrostbiteTimer(partHealth, 3)

		cond := partHealth.GetCondition(gc.ConditionFrostbite)
		require.NotNil(t, cond)
		assert.Equal(t, 0.25, cond.Timer) // delta=0.25
	})

	t.Run("やや寒い環境では凍傷タイマーが変化しない", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:  gc.ConditionFrostbite,
			Timer: 50,
		})

		// 有効温度6-10度は現状維持
		updateFrostbiteTimer(partHealth, 8)

		cond := partHealth.GetCondition(gc.ConditionFrostbite)
		require.NotNil(t, cond)
		assert.Equal(t, 50.0, cond.Timer) // 変化なし
	})

	t.Run("暖かい環境で凍傷タイマーが回復", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:  gc.ConditionFrostbite,
			Timer: 50,
		})

		updateFrostbiteTimer(partHealth, 20)

		cond := partHealth.GetCondition(gc.ConditionFrostbite)
		require.NotNil(t, cond)
		assert.Equal(t, 49.75, cond.Timer) // -0.25 回復
	})

	t.Run("タイマーが0になると状態が削除される", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:  gc.ConditionFrostbite,
			Timer: 0.25,
		})

		updateFrostbiteTimer(partHealth, 20)

		cond := partHealth.GetCondition(gc.ConditionFrostbite)
		assert.Nil(t, cond) // 削除されたはず
	})
}

func TestGetWorstSeverity(t *testing.T) {
	t.Parallel()

	t.Run("状態がない場合はSeverityNone", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		assert.Equal(t, gc.SeverityNone, getWorstSeverity(hs))
	})

	t.Run("最も重いSeverityを返す", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartTorso].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityMinor,
		})
		hs.Parts[gc.BodyPartHead].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
		})

		assert.Equal(t, gc.SeveritySevere, getWorstSeverity(hs))
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

		// プレイヤーエンティティを作成
		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		sys := &TemperatureSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		// 寒い環境なので低体温のタイマーが増加しているはず
		hs := world.Components.HealthStatus.Get(player).(*gc.HealthStatus)
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
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
		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		insulation := CalculateEquippedInsulation(world, player)

		for i := 0; i < int(gc.BodyPartCount); i++ {
			assert.Equal(t, 0, insulation[i].Cold)
			assert.Equal(t, 0, insulation[i].Heat)
		}
	})

	t.Run("装備の断熱値が反映される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		// 胴体装備を作成（耐寒10, 耐熱5）
		armor := world.Manager.NewEntity()
		armor.AddComponent(world.Components.Wearable, &gc.Wearable{
			EquipmentCategory: gc.EquipmentTorso,
			InsulationCold:    10,
			InsulationHeat:    5,
		})
		armor.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{
			Owner: player,
		})

		insulation := CalculateEquippedInsulation(world, player)

		// 胴体に適用されているはず
		assert.Equal(t, 10, insulation[gc.BodyPartTorso].Cold)
		assert.Equal(t, 5, insulation[gc.BodyPartTorso].Heat)
		// 頭には適用されていないはず
		assert.Equal(t, 0, insulation[gc.BodyPartHead].Cold)
		assert.Equal(t, 0, insulation[gc.BodyPartHead].Heat)
	})
}

func TestCalculateHypothermiaEffects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		part         gc.BodyPart
		severity     gc.Severity
		expectedLen  int
		expectedStat gc.StatType
	}{
		{"胴体-軽度", gc.BodyPartTorso, gc.SeverityMinor, 2, gc.StatStrength},
		{"胴体-中度", gc.BodyPartTorso, gc.SeverityMedium, 2, gc.StatStrength},
		{"胴体-重度", gc.BodyPartTorso, gc.SeveritySevere, 2, gc.StatStrength},
		{"頭-軽度", gc.BodyPartHead, gc.SeverityMinor, 1, gc.StatSensation},
		{"腕-軽度", gc.BodyPartArms, gc.SeverityMinor, 1, gc.StatStrength},
		{"手-軽度", gc.BodyPartHands, gc.SeverityMinor, 1, gc.StatDexterity},
		{"脚-軽度", gc.BodyPartLegs, gc.SeverityMinor, 1, gc.StatAgility},
		{"足-軽度", gc.BodyPartFeet, gc.SeverityMinor, 1, gc.StatAgility},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			effects := calculateHypothermiaEffects(tt.part, tt.severity)
			require.Len(t, effects, tt.expectedLen)
			assert.Equal(t, tt.expectedStat, effects[0].Stat)
			assert.Less(t, effects[0].Value, 0) // 効果は負の値
		})
	}

	t.Run("SeverityNoneでは効果なし", func(t *testing.T) {
		t.Parallel()
		effects := calculateHypothermiaEffects(gc.BodyPartTorso, gc.SeverityNone)
		assert.Nil(t, effects)
	})

	t.Run("胴体の低体温はStrengthとVitalityを下げる", func(t *testing.T) {
		t.Parallel()
		effects := calculateHypothermiaEffects(gc.BodyPartTorso, gc.SeverityMinor)
		require.Len(t, effects, 2)
		assert.Equal(t, gc.StatStrength, effects[0].Stat)
		assert.Equal(t, gc.StatVitality, effects[1].Stat)
	})

	t.Run("重度の方が効果が大きい", func(t *testing.T) {
		t.Parallel()
		minorEffects := calculateHypothermiaEffects(gc.BodyPartTorso, gc.SeverityMinor)
		severeEffects := calculateHypothermiaEffects(gc.BodyPartTorso, gc.SeveritySevere)

		// 重度の効果値の絶対値が大きい
		assert.Greater(t, -severeEffects[0].Value, -minorEffects[0].Value)
	})
}

func TestCalculateHyperthermiaEffects(t *testing.T) {
	t.Parallel()

	t.Run("胴体の高体温はStrengthを下げる", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartTorso, gc.SeverityMinor)
		require.Len(t, effects, 1)
		assert.Equal(t, gc.StatStrength, effects[0].Stat)
		assert.Equal(t, -1, effects[0].Value)
	})

	t.Run("頭の高体温はSensationを下げる", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartHead, gc.SeverityMinor)
		require.Len(t, effects, 1)
		assert.Equal(t, gc.StatSensation, effects[0].Stat)
		assert.Equal(t, -1, effects[0].Value)
	})

	t.Run("手は高体温の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartHands, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("足は高体温の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartFeet, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("腕は高体温の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartArms, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("脚は高体温の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartLegs, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("SeverityNoneでは効果なし", func(t *testing.T) {
		t.Parallel()
		effects := calculateHyperthermiaEffects(gc.BodyPartTorso, gc.SeverityNone)
		assert.Nil(t, effects)
	})

	t.Run("重度の方が効果が大きい", func(t *testing.T) {
		t.Parallel()
		minorEffects := calculateHyperthermiaEffects(gc.BodyPartTorso, gc.SeverityMinor)
		severeEffects := calculateHyperthermiaEffects(gc.BodyPartTorso, gc.SeveritySevere)

		assert.Equal(t, -1, minorEffects[0].Value)
		assert.Equal(t, -3, severeEffects[0].Value)
	})
}

func TestCalculateFrostbiteEffects(t *testing.T) {
	t.Parallel()

	t.Run("手の凍傷はDexterityを下げる", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartHands, gc.SeverityMinor)
		require.Len(t, effects, 1)
		assert.Equal(t, gc.StatDexterity, effects[0].Stat)
		assert.Equal(t, -1, effects[0].Value)
	})

	t.Run("足の凍傷はAgilityを下げる", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartFeet, gc.SeverityMinor)
		require.Len(t, effects, 1)
		assert.Equal(t, gc.StatAgility, effects[0].Stat)
		assert.Equal(t, -1, effects[0].Value)
	})

	t.Run("胴体は凍傷の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartTorso, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("頭は凍傷の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartHead, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("腕は凍傷の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartArms, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("脚は凍傷の影響を受けない", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartLegs, gc.SeveritySevere)
		assert.Nil(t, effects)
	})

	t.Run("SeverityNoneでは効果なし", func(t *testing.T) {
		t.Parallel()
		effects := calculateFrostbiteEffects(gc.BodyPartHands, gc.SeverityNone)
		assert.Nil(t, effects)
	})

	t.Run("重度の方が効果が大きい", func(t *testing.T) {
		t.Parallel()
		minorEffects := calculateFrostbiteEffects(gc.BodyPartHands, gc.SeverityMinor)
		severeEffects := calculateFrostbiteEffects(gc.BodyPartHands, gc.SeveritySevere)

		assert.Equal(t, -1, minorEffects[0].Value)
		assert.Equal(t, -3, severeEffects[0].Value)
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

		updateConditionEffects(partHealth, gc.BodyPartTorso)

		cond := partHealth.GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		require.Len(t, cond.Effects, 2) // Strength と Vitality
	})

	t.Run("高体温の効果が適用される", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHyperthermia,
			Severity: gc.SeverityMedium,
			Timer:    50,
		})

		updateConditionEffects(partHealth, gc.BodyPartHead)

		cond := partHealth.GetCondition(gc.ConditionHyperthermia)
		require.NotNil(t, cond)
		require.Len(t, cond.Effects, 1) // Sensation
		assert.Equal(t, gc.StatSensation, cond.Effects[0].Stat)
	})

	t.Run("凍傷の効果が適用される", func(t *testing.T) {
		t.Parallel()
		partHealth := &gc.BodyPartHealth{}
		partHealth.SetCondition(gc.HealthCondition{
			Type:     gc.ConditionFrostbite,
			Severity: gc.SeveritySevere,
			Timer:    80,
		})

		updateConditionEffects(partHealth, gc.BodyPartHands)

		cond := partHealth.GetCondition(gc.ConditionFrostbite)
		require.NotNil(t, cond)
		require.Len(t, cond.Effects, 1) // Dexterity
		assert.Equal(t, gc.StatDexterity, cond.Effects[0].Stat)
		assert.Equal(t, -3, cond.Effects[0].Value) // Severe = multiplier 3
	})
}

func TestLogTemperatureChange(t *testing.T) {
	t.Parallel()

	t.Run("悪化時のメッセージが取得できる", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeverityMinor)
		assert.Contains(t, msg, "冷えてきた")
		assert.Contains(t, msg, "[胴体]")
	})

	t.Run("中程度悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeverityMedium)
		assert.Contains(t, msg, "かなり冷えている")
	})

	t.Run("重度悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeveritySevere)
		assert.Contains(t, msg, "危険な状態")
	})

	t.Run("高体温悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.BodyPartHead, gc.ConditionHyperthermia, gc.SeverityMinor)
		assert.Contains(t, msg, "火照ってきた")
		assert.Contains(t, msg, "[頭]")
	})

	t.Run("凍傷悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.BodyPartHands, gc.ConditionFrostbite, gc.SeverityMinor)
		assert.Contains(t, msg, "凍傷になりかけている")
	})

	t.Run("回復時のメッセージが取得できる", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeverityNone)
		assert.Contains(t, msg, "温まった")
	})

	t.Run("部分回復のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeverityMinor)
		assert.Contains(t, msg, "少し温まってきた")
	})

	t.Run("高体温回復のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.BodyPartHead, gc.ConditionHyperthermia, gc.SeverityNone)
		assert.Contains(t, msg, "涼しくなった")
	})

	t.Run("凍傷回復のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.BodyPartHands, gc.ConditionFrostbite, gc.SeverityNone)
		assert.Contains(t, msg, "凍傷が治った")
	})

	t.Run("SeverityNoneの悪化メッセージは空", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeverityNone)
		assert.Empty(t, msg)
	})

	t.Run("SeveritySevereの回復メッセージは空", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.BodyPartTorso, gc.ConditionHypothermia, gc.SeveritySevere)
		assert.Empty(t, msg)
	})
}
