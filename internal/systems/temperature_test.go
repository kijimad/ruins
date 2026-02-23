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

		warmth := [gc.BodyPartCount]int{}
		updateTemperatureConditions(hs, 20, warmth, false)

		// タイマーが減少しているはず
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
		if cond != nil {
			assert.Less(t, cond.Timer, 50.0)
		}
	})

	t.Run("寒い環境で低体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		warmth := [gc.BodyPartCount]int{}

		// 非常に寒い環境
		updateTemperatureConditions(hs, 0, warmth, false)

		// 低体温の状態が追加されているはず
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("暑い環境で高体温タイマーが増加", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		warmth := [gc.BodyPartCount]int{}

		// 非常に暑い環境
		updateTemperatureConditions(hs, 40, warmth, false)

		// 高体温の状態が追加されているはず
		cond := hs.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHyperthermia)
		require.NotNil(t, cond)
		assert.Greater(t, cond.Timer, 0.0)
	})

	t.Run("装備保温値で有効温度が上がる", func(t *testing.T) {
		t.Parallel()
		hs1 := &gc.HealthStatus{}
		hs2 := &gc.HealthStatus{}
		noWarmth := [gc.BodyPartCount]int{}
		withWarmth := [gc.BodyPartCount]int{}
		withWarmth[gc.BodyPartTorso] = 20 // 保温+20

		// 同じ寒い環境(0度)で比較
		updateTemperatureConditions(hs1, 0, noWarmth, false)
		updateTemperatureConditions(hs2, 0, withWarmth, false)

		// 保温ありの方がタイマー増加が少ないはず
		cond1 := hs1.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)
		cond2 := hs2.Parts[gc.BodyPartTorso].GetCondition(gc.ConditionHypothermia)

		// 保温なしは状態が追加される
		require.NotNil(t, cond1)
		assert.Greater(t, cond1.Timer, 0.0)

		// 保温20で有効温度が20度になり、快適範囲なので状態は追加されないか軽微
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
		warmth := [gc.BodyPartCount]int{}

		// 非常に寒い環境（delta=+0.5）でSeverityがNone→Minorに変化
		hasChange := updateTemperatureConditions(hs, 0, warmth, false)
		assert.True(t, hasChange)
	})

	t.Run("Severity変化がない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		hs := &gc.HealthStatus{}
		warmth := [gc.BodyPartCount]int{}

		// 快適な温度で初期状態ならSeverity変化なし
		hasChange := updateTemperatureConditions(hs, 20, warmth, false)
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

func TestComfortableTemp(t *testing.T) {
	t.Parallel()

	t.Run("快適温度は20度", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 20, comfortableTemp)
	})
}
