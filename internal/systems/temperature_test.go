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

// coldDungeonName は基本気温0度のテスト用ダンジョン定義名。DungeonForest の英語 id。
const coldDungeonName = "Dead forest"

func TestGetTileTemperatureAt(t *testing.T) {
	t.Parallel()

	t.Run("タイルが存在する場合は気温修正を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		tile := world.ECS.NewEntity()
		world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
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

func TestAmbientTemperatureAt_オーバーワールドは季節の世界温度そのもの(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 25日目の昼。冬の底 -30 に昼補正 +10
	query.GetGameTime(world).TotalTurns = 24 * 1500
	// 帯データの有無が屋外判定を兼ねる。オーバーワールドにして帯を付ける
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)

	temp, err := AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -20, temp, "屋外は季節世界温度 -30 に昼補正 +10 を足す")
}

func TestAmbientTemperatureAt_屋外はステージ定義が無くても世界温度を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 25日目の昼。世界温度 = -30 + 10 = -20
	query.GetGameTime(world).TotalTurns = 24 * 1500
	// 定義が登録されていない屋外ステージ。帯を付けて屋外と判定させる
	query.GetDungeon(world).CurrentStage = gc.StageKey{Name: "未登録の屋外"}
	query.EnsureSeamlessBand(world)

	temp, err := AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -20, temp, "屋外はステージ定義に依らず世界温度そのものを返す")
}

func TestAmbientTemperatureAt_ダンジョンは世界温度を緩和して受ける(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 25日目の昼。世界温度 = -30 + 10 = -20。屋内は基本気温0に世界温度の半分を足す
	query.GetGameTime(world).TotalTurns = 24 * 1500
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)

	temp, err := AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -10, temp, "屋内は基本気温 0 に世界温度 -20 の半分 -10 を足す")
}

func TestAmbientTemperatureAt_冬の屋内は屋外より暖かい(t *testing.T) {
	t.Parallel()
	// 冬の底の同じ時刻で、屋内と屋外の周囲気温を比べる。屋内が屋外より暖かく寒さの逆転がない
	const winterNoon consts.Turn = 24 * 1500

	outdoor := func() int {
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = winterNoon
		query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
		query.EnsureSeamlessBand(world)
		temp, err := AmbientTemperatureAt(world, 0, 0)
		require.NoError(t, err)
		return temp
	}()
	indoor := func() int {
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = winterNoon
		query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)
		temp, err := AmbientTemperatureAt(world, 0, 0)
		require.NoError(t, err)
		return temp
	}()

	assert.Greater(t, indoor, outdoor, "冬でも屋内は屋外より暖かい退避先になる")
}

func TestCalcBodyTempRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		effectiveTemp int
		expected      float64
	}{
		{"極寒(-50度以下)", -100, -0.5},
		{"極寒(-50度)", -50, -0.5},
		{"非常に寒い(-49度)", -49, -0.2},
		{"非常に寒い(0度以下)", -10, -0.2},
		{"非常に寒い(0度)", 0, -0.2},
		{"寒い(1-10度)", 5, -0.1},
		{"寒い(10度)", 10, -0.1},
		{"やや寒い(11-15度)", 12, 0},
		{"やや寒い(15度)", 15, 0},
		{"快適(16-25度)", 20, 0},
		{"快適(25度)", 25, 0},
		{"適温(26-30度)", 28, 0},
		{"適温(30度)", 30, 0},
		{"暑くても害はない(33度)", 33, 0},
		{"暑くても害はない(35度)", 35, 0},
		{"非常に暑くても害はない(40度)", 40, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := calcBodyTempRate(tt.effectiveTemp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateTemperatureConditions(t *testing.T) {
	t.Parallel()

	t.Run("体温が正常帯なら状態が回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:  gc.ConditionHypothermia,
			Timer: 50,
		})

		updateTemperatureConditions(world, hs, false, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.InDelta(t, 49.75, cond.Timer, 1e-9)
	})

	t.Run("体温が正常帯を下回ると低体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{BodyTempOffset: -3.0}

		updateTemperatureConditions(world, hs, false, 100)

		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.InDelta(t, 0.5, cond.Timer, 1e-9, "帯の縁0.25に超過1°Cぶん0.25を足す")
	})

	t.Run("超過が深いほど速く進み上限で頭打ちになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		shallow := &gc.HealthStatus{BodyTempOffset: -2.5}
		deep := &gc.HealthStatus{BodyTempOffset: -5.0}

		updateTemperatureConditions(world, shallow, false, 100)
		updateTemperatureConditions(world, deep, false, 100)

		shallowTimer := shallow.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia).Timer
		deepTimer := deep.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia).Timer
		assert.Less(t, shallowTimer, deepTimer)
		assert.InDelta(t, 1.0, deepTimer, 1e-9, "超過3°Cで上限1.0に達する")
	})

	t.Run("Severity変化時にtrueを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{BodyTempOffset: -3.0}
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeverityNone,
			Timer:    24.75,
		})

		hasChange := updateTemperatureConditions(world, hs, false, 100)
		assert.True(t, hasChange)
	})

	t.Run("Severity変化がない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hs := &gc.HealthStatus{}

		hasChange := updateTemperatureConditions(world, hs, false, 100)
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

	t.Run("寒い環境で体温が下がり続けると低体温に至る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1) // 基本気温0度

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)

		sys := &TemperatureSystem{}
		require.NoError(t, sys.Update(world))

		// まず体温が下がり、正常帯の内は状態が付かない
		hs := world.Components.HealthStatus.Get(player)
		assert.Negative(t, hs.BodyTempOffset)
		assert.Nil(t, hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia), "正常帯の内はタイマーが進まない")

		// 冷やし続けると正常帯を割って低体温が付く
		for range 25 {
			require.NoError(t, sys.Update(world))
		}
		cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("存在しないダンジョン名の場合はエラーなし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetDungeon(world).CurrentStage = gc.NewDungeonStage("存在しないダンジョン", 1)

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
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)

		insulation := CalculateEquippedInsulation(world, player)
		assert.Equal(t, 0, insulation.Cold)
		assert.Equal(t, 0, insulation.Heat)
	})

	t.Run("装備の断熱値が合算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
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
}

func TestLogTemperatureChange(t *testing.T) {
	t.Parallel()

	t.Run("悪化時のメッセージが取得できる", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeverityMinor)
		assert.Contains(t, msg, "The cold is setting in")
	})

	t.Run("中程度悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeverityMedium)
		assert.Contains(t, msg, "You are quite cold")
	})

	t.Run("重度悪化のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getWorseningMessage(gc.ConditionHypothermia, gc.SeveritySevere)
		assert.Contains(t, msg, "The cold is dangerous")
	})

	t.Run("回復時のメッセージが取得できる", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.ConditionHypothermia, gc.SeverityNone)
		assert.Contains(t, msg, "You have warmed up")
	})

	t.Run("部分回復のメッセージ", func(t *testing.T) {
		t.Parallel()
		msg := getRecoveryMessage(gc.ConditionHypothermia, gc.SeverityMinor)
		assert.Contains(t, msg, "You are warming up a little")
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

func TestHeatSourceWarmthAt_距離に応じて減衰し半径外は無視する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	addHeatSource := func(x, y, radius consts.Tile, warmth float64) {
		e := world.ECS.NewEntity()
		world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		world.Components.HeatSource.Add(e, &gc.HeatSource{Radius: radius, Warmth: warmth})
		// Burning を持つ熱源だけが暖房として数えられる
		world.Components.Burning.Add(e, &gc.Burning{Remaining: 1})
	}
	addHeatSource(5, 5, 2, 0.6)
	addHeatSource(20, 20, 1, 9.9) // 遠く、どの検証点からも圏外

	// 源泉は満額、距離1は 2/3、距離2は 1/3 と線形に減衰する
	assert.InDelta(t, 0.6, heatSourceWarmthAt(world, 5, 5), 1e-9)
	assert.InDelta(t, 0.4, heatSourceWarmthAt(world, 6, 5), 1e-9)
	assert.InDelta(t, 0.2, heatSourceWarmthAt(world, 7, 5), 1e-9)
	assert.InDelta(t, 0.0, heatSourceWarmthAt(world, 8, 5), 1e-9)
}

func TestHeatSourceWarmthAt_複数の熱源を加算する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	addHeatSource := func(x, y, radius consts.Tile, warmth float64) {
		e := world.ECS.NewEntity()
		world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		world.Components.HeatSource.Add(e, &gc.HeatSource{Radius: radius, Warmth: warmth})
		// Burning を持つ熱源だけが暖房として数えられる
		world.Components.Burning.Add(e, &gc.Burning{Remaining: 1})
	}
	addHeatSource(5, 5, 1, 0.5)
	addHeatSource(6, 6, 2, 0.3)

	assert.InDelta(t, 0.7, heatSourceWarmthAt(world, 5, 5), 1e-9)
}

func TestTemperatureSystem_Update_熱源のそばは体温の低下が緩む(t *testing.T) {
	t.Parallel()

	run := func(withBonfire bool) float64 {
		world := testutil.InitTestWorld(t)
		query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		if withBonfire {
			// プレイヤーの隣に焚き火。raw の heatSource から熱源になる
			_, err := lifecycle.SpawnProp(world, "bonfire", 6, 5)
			require.NoError(t, err)
		}
		sys := &TemperatureSystem{}
		for range 10 {
			require.NoError(t, sys.Update(world))
		}
		return world.Components.HealthStatus.Get(player).BodyTempOffset
	}

	withFire := run(true)
	without := run(false)
	assert.Greater(t, withFire, without, "熱源のそばは体温の低下が緩む")
}

func TestBodyTempRate_寒い環境では負になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
	require.NoError(t, err)

	assert.Negative(t, bodyTempRate(world, player), "寒い環境では体温が下降する")
}

func TestBodyTempRate_冷えた体は熱源のそばで温まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	world.Components.HealthStatus.Get(player).BodyTempOffset = -3.0

	// 環境の冷えを上回る暖かさになるよう焚き火を2つ隣接させる
	_, err = lifecycle.SpawnProp(world, "bonfire", 6, 5)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "bonfire", 4, 5)
	require.NoError(t, err)

	assert.Positive(t, bodyTempRate(world, player), "熱源の暖かさが冷えを上回れば体温が上昇する")
}

func TestBodyTempRate_平熱以上では熱源が効かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage("Debug town", 1)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnProp(world, "bonfire", 6, 5)
	require.NoError(t, err)

	assert.Zero(t, bodyTempRate(world, player), "平熱の体は熱源で温まらない")
}

func TestBodyTempRate_外因が無ければ平熱へ戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage("Debug town", 1)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	hs := world.Components.HealthStatus.Get(player)

	hs.BodyTempOffset = -1.0
	assert.InDelta(t, 0.1, bodyTempRate(world, player), 1e-9, "冷えていれば平熱へ向けて上がる")

	hs.BodyTempOffset = -0.05
	assert.InDelta(t, 0.05, bodyTempRate(world, player), 1e-9, "残りが小さければ平熱ちょうどで止まる")
}
