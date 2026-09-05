package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// coldDungeonName は基本気温0度のテスト用ダンジョン定義名。DungeonForest の英語 id。
const coldDungeonName = "Dead forest"

// setStage は現ステージを切り替え、生成時と同じく基本気温を StageField へ写す。
// 気温計算は StageField.BaseTemp を読むので、テストでも生成の手順を再現する
func setStage(world w.World, key gc.StageKey) {
	query.GetDungeon(world).CurrentStage = key
	query.EnsureStageField(world, key).BaseTemp = dungeon.BaseTemperatureFor(key.Name)
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
		setStage(world, gc.NewDungeonStage(coldDungeonName, 1)) // 基本気温0度

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
		setStage(world, gc.NewDungeonStage("存在しないダンジョン", 1))

		sys := &TemperatureSystem{}
		err := sys.Update(world)

		assert.NoError(t, err)
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

func TestTemperatureSystem_Update_熱源のそばは体温の低下が緩む(t *testing.T) {
	t.Parallel()

	run := func(withBonfire bool) float64 {
		world := testutil.InitTestWorld(t)
		setStage(world, gc.NewDungeonStage(coldDungeonName, 1))
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		if withBonfire {
			// プレイヤーの隣に焚き火。raw の heatSource から熱源になる
			_, err := lifecycle.SpawnProp(world, "fire", 6, 5)
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
	setStage(world, gc.NewDungeonStage(coldDungeonName, 1))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
	require.NoError(t, err)

	assert.Negative(t, bodyTempRate(world, player), "寒い環境では体温が下降する")
}

func TestBodyTempRate_冷えた体は熱源のそばで温まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	setStage(world, gc.NewDungeonStage(coldDungeonName, 1))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	world.Components.HealthStatus.Get(player).BodyTempOffset = -3.0

	// 環境の冷えを上回る暖かさになるよう焚き火を2つ隣接させる
	_, err = lifecycle.SpawnProp(world, "fire", 6, 5)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "fire", 4, 5)
	require.NoError(t, err)

	assert.Positive(t, bodyTempRate(world, player), "熱源の暖かさが冷えを上回れば体温が上昇する")
}

func TestBodyTempRate_平熱以上では熱源が効かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	setStage(world, gc.NewDungeonStage("Debug town", 1))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	_, err = lifecycle.SpawnProp(world, "fire", 6, 5)
	require.NoError(t, err)

	assert.Zero(t, bodyTempRate(world, player), "平熱の体は熱源で温まらない")
}

func TestBodyTempRate_外因が無ければ平熱へ戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	setStage(world, gc.NewDungeonStage("Debug town", 1))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	hs := world.Components.HealthStatus.Get(player)

	hs.BodyTempOffset = -1.0
	assert.InDelta(t, 0.1, bodyTempRate(world, player), 1e-9, "冷えていれば平熱へ向けて上がる")

	hs.BodyTempOffset = -0.05
	assert.InDelta(t, 0.05, bodyTempRate(world, player), 1e-9, "残りが小さければ平熱ちょうどで止まる")
}

func TestTemperatureSystem_重症低体温はHPを削る(t *testing.T) {
	t.Parallel()

	// Timer を 100 にすると1ターンの体温変動後も必ず Severe に留まるので、環境に依らず判定できる。
	setHypothermiaTimer := func(hs *gc.HealthStatus, timer float64) {
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.TimerToSeverity(timer),
			Timer:    timer,
		})
	}

	t.Run("重症の低体温は失血でHPを削る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		setHypothermiaTimer(world.Components.HealthStatus.Get(player), 100)
		hpBefore := world.Components.HP.Get(player).Current

		// 低体温は直接でなく血液量を下げて殺す。適用は ConditionSystem
		require.NoError(t, (&ConditionSystem{}).Update(world))

		assert.Less(t, world.Components.HP.Get(player).Current, hpBefore, "重症の低体温は失血で HP を削る")
	})

	t.Run("中度では減らない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		setHypothermiaTimer(world.Components.HealthStatus.Get(player), 60)
		hpBefore := world.Components.HP.Get(player).Current

		require.NoError(t, (&ConditionSystem{}).Update(world))

		assert.Equal(t, hpBefore, world.Components.HP.Get(player).Current, "中度は血液量を下げないので削られない")
	})
}
