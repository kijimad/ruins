package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// coldDungeonName は基本気温0度のテスト用ダンジョン定義名。
const coldDungeonName = "亡者の森"

func TestGetTileTemperatureAt(t *testing.T) {
	t.Parallel()

	t.Run("タイルが存在する場合は気温修正を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		tile := world.ECS.NewEntity()
		world.Components.GridElement.Add(tile, &gc.GridElement{X: 5, Y: 5})
		world.Components.TileTemperature.Add(tile, &gc.TileTemperature{
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

func TestFrostZoneModifier(t *testing.T) {
	t.Parallel()

	t.Run("極低温ゾーン内のタイルに極寒修正を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sb := &query.GetDungeon(world).SeamlessBand
		sb.Front.Active = true
		sb.EastIndex = 0
		sb.ChunkW = 40
		sb.Front.ColdWidth = 20
		sb.Front.EastAbsX = 30 // ゾーンは半開区間 (10, 30]

		assert.Equal(t, 0, frostZoneModifier(world, 10), "西端は含まない（進入不可ライン）")
		assert.Equal(t, FrostZoneTempModifier, frostZoneModifier(world, 11), "ゾーン内は極寒")
		assert.Equal(t, FrostZoneTempModifier, frostZoneModifier(world, 30), "東端は含む")
		assert.Equal(t, 0, frostZoneModifier(world, 31), "前線より東は平常")
	})

	t.Run("帯原点で絶対Xに変換して判定する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sb := &query.GetDungeon(world).SeamlessBand
		sb.Front.Active = true
		sb.EastIndex = 1 // bandOriginX = 1*40 = 40
		sb.ChunkW = 40
		sb.Front.ColdWidth = 20
		sb.Front.EastAbsX = 60 // ゾーン (40, 60]。ローカル x=10 → absX=50 は内側

		assert.Equal(t, FrostZoneTempModifier, frostZoneModifier(world, 10), "ローカル10=絶対50はゾーン内")
		assert.Equal(t, 0, frostZoneModifier(world, 25), "ローカル25=絶対65はゾーン外")
	})

	t.Run("FrontActiveでないと無効", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sb := &query.GetDungeon(world).SeamlessBand
		sb.Front.Active = false
		sb.Front.EastAbsX = 30
		sb.Front.ColdWidth = 20
		assert.Equal(t, 0, frostZoneModifier(world, 20), "通常ダンジョンでは前線無効")
	})
}

func TestCalculateEnvTemperature_極低温ゾーンで極寒になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.DefinitionName = coldDungeonName // 基本気温0度
	sb := &d.SeamlessBand
	sb.Front.Active = true
	sb.EastIndex = 0
	sb.ChunkW = 40
	sb.Front.ColdWidth = 20
	sb.Front.EastAbsX = 30 // ゾーン (10, 30]

	inZone, err := CalculateEnvTemperature(world, 20, 0)
	require.NoError(t, err)
	outZone, err := CalculateEnvTemperature(world, 35, 0)
	require.NoError(t, err)

	assert.Equal(t, FrostZoneTempModifier, inZone-outZone, "ゾーン内はゾーン外より極寒修正ぶん低い")
	assert.LessOrEqual(t, inZone, 0, "ゾーン内は最大寒冷（0度以下）になり低体温が急進する")
}

func TestTemperatureSystem_極低温ゾーンで低体温が急進する(t *testing.T) {
	t.Parallel()

	// front=true なら極低温ゾーン内、false なら同じ位置の通常環境でプレイヤーを1ターン更新する。
	setup := func(front bool) *gc.HealthStatus {
		world := testutil.InitTestWorld(t)
		d := query.GetDungeon(world)
		d.DefinitionName = coldDungeonName // 基本気温0度
		if front {
			sb := &d.SeamlessBand
			sb.Front.Active = true
			sb.EastIndex = 0
			sb.ChunkW = 40
			sb.Front.ColdWidth = 20
			sb.Front.EastAbsX = 30 // ゾーン (10, 30]。プレイヤー x=20 は内側
		}
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 20, Y: 0}, "Ash")
		require.NoError(t, err)
		require.NoError(t, (&TemperatureSystem{}).Update(world))
		return world.Components.HealthStatus.Get(player)
	}

	inZone := setup(true).Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
	require.NotNil(t, inZone, "ゾーン内では低体温が発生する")

	// 通常環境は快適で低体温が付かないこともあるので nil を 0 として扱う
	normalCold := setup(false).Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
	normalTimer := 0.0
	if normalCold != nil {
		normalTimer = normalCold.Timer
	}

	assert.Greater(t, inZone.Timer, normalTimer, "極低温ゾーンは通常の寒さより低体温が速く進む")
}

func TestCalcTimerDelta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		effectiveTemp int
		expected      float64
	}{
		{"極寒(-50度以下)", -100, -1.0},
		{"極寒(-50度)", -50, -1.0},
		{"非常に寒い(-49度)", -49, -0.5},
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
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:  gc.ConditionHypothermia,
			Timer: 50,
		})

		updateTemperatureConditions(world, hs, 20, Insulation{}, false, 100, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		if cond != nil {
			assert.Less(t, cond.Timer, 50.0)
		}
	})

	t.Run("寒い環境で低体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}

		updateTemperatureConditions(world, hs, 0, Insulation{}, false, 100, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("暑い環境で高体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}

		updateTemperatureConditions(world, hs, 40, Insulation{}, false, 100, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHyperthermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("耐寒装備で低体温を軽減", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs1 := &gc.HealthStatus{}
		hs2 := &gc.HealthStatus{}

		// 同じ寒い環境(0度)で比較
		updateTemperatureConditions(world, hs1, 0, Insulation{}, false, 100, 100)
		updateTemperatureConditions(world, hs2, 0, Insulation{Cold: 20}, false, 100, 100)

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
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityNone,
			Timer:    24.5,
		})

		hasChange := updateTemperatureConditions(world, hs, 0, Insulation{}, false, 100, 100)
		assert.True(t, hasChange)
	})

	t.Run("Severity変化がない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}

		hasChange := updateTemperatureConditions(world, hs, 20, Insulation{}, false, 100, 100)
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
		query.SetDungeon(world, nil)

		sys := &TemperatureSystem{}
		err := sys.Update(world)

		assert.Error(t, err)
	})

	t.Run("HealthStatusを持つエンティティの状態が更新される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetDungeon(world).DefinitionName = coldDungeonName // 基本気温0度

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "Ash")
		require.NoError(t, err)

		sys := &TemperatureSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		// 寒い環境なので低体温のタイマーが増加しているはず
		hs := world.Components.HealthStatus.Get(player)
		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("存在しないダンジョン名の場合はエラーなし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetDungeon(world).DefinitionName = "存在しないダンジョン"

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
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "Ash")
		require.NoError(t, err)

		insulation := CalculateEquippedInsulation(world, player)
		assert.Equal(t, 0, insulation.Cold)
		assert.Equal(t, 0, insulation.Heat)
	})

	t.Run("装備の断熱値が合算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "Ash")
		require.NoError(t, err)

		// 胴体装備（耐寒10, 耐熱5）
		armor := world.ECS.NewEntity()
		world.Components.Wearable.Add(armor, &gc.Wearable{
			EquipmentCategory: gc.EquipmentTorso,
			InsulationCold:    10,
			InsulationHeat:    5,
		})
		world.Components.LocationEquipped.Add(armor, &gc.LocationEquipped{
			Owner: player,
		})

		// 頭装備（耐寒3, 耐熱2）
		helmet := world.ECS.NewEntity()
		world.Components.Wearable.Add(helmet, &gc.Wearable{
			EquipmentCategory: gc.EquipmentHead,
			InsulationCold:    3,
			InsulationHeat:    2,
		})
		world.Components.LocationEquipped.Add(helmet, &gc.LocationEquipped{
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
