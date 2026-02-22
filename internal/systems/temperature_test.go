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

func TestUpdateBodyTemperature(t *testing.T) {
	t.Parallel()

	t.Run("環境気温20度で正常体温を維持", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()

		updateBodyTemperature(bt, 20)

		for i := 0; i < int(gc.BodyPartCount); i++ {
			assert.Equal(t, gc.TempNormal, bt.Parts[i].Convergent, "収束温度が正常体温でない")
			assert.Equal(t, gc.TempNormal, bt.Parts[i].Temp, "現在温度が正常体温でない")
		}
	})

	t.Run("環境気温0度で収束温度が下がる", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()

		updateBodyTemperature(bt, 0)

		// 収束温度 = 50 + (0 - 20) * 2 = 50 - 40 = 10
		for i := 0; i < int(gc.BodyPartCount); i++ {
			assert.Equal(t, 10, bt.Parts[i].Convergent, "収束温度が10でない")
		}
	})

	t.Run("環境気温40度で収束温度が上がる", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()

		updateBodyTemperature(bt, 40)

		// 収束温度 = 50 + (40 - 20) * 2 = 50 + 40 = 90
		for i := 0; i < int(gc.BodyPartCount); i++ {
			assert.Equal(t, 90, bt.Parts[i].Convergent, "収束温度が90でない")
		}
	})

	t.Run("収束温度は0-100にクランプされる", func(t *testing.T) {
		t.Parallel()

		// 非常に寒い環境
		bt1 := gc.NewBodyTemperature()
		updateBodyTemperature(bt1, -50)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			assert.GreaterOrEqual(t, bt1.Parts[i].Convergent, 0, "収束温度が0未満")
		}

		// 非常に暑い環境
		bt2 := gc.NewBodyTemperature()
		updateBodyTemperature(bt2, 100)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			assert.LessOrEqual(t, bt2.Parts[i].Convergent, 100, "収束温度が100超過")
		}
	})

	t.Run("体温は収束温度に向かって変化する", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		initialTemp := bt.Parts[gc.BodyPartTorso].Temp

		// 寒い環境で何度か更新
		for i := 0; i < 10; i++ {
			updateBodyTemperature(bt, 0)
		}

		assert.Less(t, bt.Parts[gc.BodyPartTorso].Temp, initialTemp, "体温が下がっていない")
	})
}

func TestUpdateFrostbiteTimer(t *testing.T) {
	t.Parallel()

	t.Run("凍結時にタイマーが速く増加", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		bt.Parts[gc.BodyPartHands].Temp = gc.TempFreezing

		initialTimer := bt.Parts[gc.BodyPartHands].FrostbiteTimer
		updateFrostbiteTimer(bt, gc.BodyPartHands)

		assert.Equal(t, initialTimer+5, bt.Parts[gc.BodyPartHands].FrostbiteTimer)
	})

	t.Run("非常に寒い時にタイマーが増加", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		bt.Parts[gc.BodyPartHands].Temp = gc.TempVeryCold

		initialTimer := bt.Parts[gc.BodyPartHands].FrostbiteTimer
		updateFrostbiteTimer(bt, gc.BodyPartHands)

		assert.Equal(t, initialTimer+3, bt.Parts[gc.BodyPartHands].FrostbiteTimer)
	})

	t.Run("寒い時にタイマーがゆっくり増加", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		bt.Parts[gc.BodyPartHands].Temp = gc.TempCold

		initialTimer := bt.Parts[gc.BodyPartHands].FrostbiteTimer
		updateFrostbiteTimer(bt, gc.BodyPartHands)

		assert.Equal(t, initialTimer+1, bt.Parts[gc.BodyPartHands].FrostbiteTimer)
	})

	t.Run("正常時にタイマーが回復", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		bt.Parts[gc.BodyPartHands].Temp = gc.TempNormal
		bt.Parts[gc.BodyPartHands].FrostbiteTimer = 50

		updateFrostbiteTimer(bt, gc.BodyPartHands)

		assert.Equal(t, 48, bt.Parts[gc.BodyPartHands].FrostbiteTimer)
	})

	t.Run("タイマーは0未満にならない", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		bt.Parts[gc.BodyPartHands].Temp = gc.TempNormal
		bt.Parts[gc.BodyPartHands].FrostbiteTimer = 1

		updateFrostbiteTimer(bt, gc.BodyPartHands)

		assert.Equal(t, 0, bt.Parts[gc.BodyPartHands].FrostbiteTimer)
	})

	t.Run("タイマー100で凍傷発症", func(t *testing.T) {
		t.Parallel()
		bt := gc.NewBodyTemperature()
		bt.Parts[gc.BodyPartHands].Temp = gc.TempFreezing
		bt.Parts[gc.BodyPartHands].FrostbiteTimer = 96

		updateFrostbiteTimer(bt, gc.BodyPartHands)

		assert.True(t, bt.Parts[gc.BodyPartHands].HasFrostbite)
	})
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

	t.Run("BodyTemperatureを持つエンティティの体温が更新される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon.DefinitionName = "亡者の森"

		// プレイヤーエンティティを作成
		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		sys := &TemperatureSystem{}
		err = sys.Update(world)

		require.NoError(t, err)

		// 環境気温 = 基本気温(10) + タイル修正(0) + 時間帯修正(0) = 10°C
		// 収束温度 = 正常体温(50) + (環境気温(10) - 快適温度(20)) * 2 = 30
		updatedBt := world.Components.BodyTemperature.Get(player).(*gc.BodyTemperature)
		assert.Equal(t, 30, updatedBt.Parts[gc.BodyPartTorso].Convergent)
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

func TestConvergenceRate(t *testing.T) {
	t.Parallel()

	t.Run("収束レートは約0.03", func(t *testing.T) {
		t.Parallel()
		assert.InDelta(t, 0.03, convergenceRate, 0.001)
	})
}

func TestComfortableTemp(t *testing.T) {
	t.Parallel()

	t.Run("快適温度は20度", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 20, comfortableTemp)
	})
}
