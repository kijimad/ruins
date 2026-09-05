package query_test

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

func TestTileEnvironmentAt(t *testing.T) {
	t.Parallel()

	t.Run("囲われは緩和度として返り加算℃には入らない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		tile := world.ECS.NewEntity()
		world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{
			Shelter: gc.ShelterFull,
		})

		shelter, modifier := query.TileEnvironmentAt(world, 5, 5)
		assert.Equal(t, gc.ShelterFull, shelter)
		assert.Equal(t, 0, modifier)
	})

	t.Run("水と植生は加算℃として返る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		tile := world.ECS.NewEntity()
		world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{
			Water:   gc.WaterNearby,
			Foliage: gc.FoliageForest,
		})

		shelter, modifier := query.TileEnvironmentAt(world, 5, 5)
		assert.Equal(t, gc.ShelterNone, shelter)
		assert.Equal(t, -8, modifier)
	})

	t.Run("タイルが存在しない場合は屋外かつ0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		shelter, modifier := query.TileEnvironmentAt(world, 5, 5)
		assert.Equal(t, gc.ShelterNone, shelter)
		assert.Equal(t, 0, modifier)
	})
}

func TestAmbientTemperatureAt_オーバーワールドは季節の世界温度そのもの(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 25日目の昼。冬の底 -30 に昼補正 +10
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	// 帯データの有無が屋外判定を兼ねる。オーバーワールドにして帯を付ける
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -20, temp, "屋外は季節世界温度 -30 に昼補正 +10 を足す")
}

func TestAmbientTemperatureAt_屋外はステージ定義が無くても世界温度を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.StageKey{Name: "未登録の屋外"}
	query.EnsureSeamlessBand(world)

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -20, temp, "屋外はステージ定義に依らず世界温度そのものを返す")
}

func TestAmbientTemperatureAt_ダンジョンの囲われたタイルは世界温度を緩和して受ける(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)

	tile := world.ECS.NewEntity()
	world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
	world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{Shelter: gc.ShelterFull})

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -5, temp, "屋内はアンカー10へ半分寄せた 10+(-20-10)/2 = -5 になる")
}

func TestAmbientTemperatureAt_ダンジョンの囲われていないタイルは世界温度をそのまま受ける(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -20, temp, "囲われが無ければ基本気温 0 に世界温度 -20 をそのまま足す")
}

func TestAmbientTemperatureAt_オーバーワールドの屋内タイルは世界温度を緩和して受ける(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)

	tile := world.ECS.NewEntity()
	world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
	world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{Shelter: gc.ShelterFull})

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -5, temp, "オーバーワールドの屋内はアンカー10へ半分寄せた -5 になる")
}

func TestAmbientTemperatureAt_温暖時も屋内は屋外より寒くならない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 0
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)

	tile := world.ECS.NewEntity()
	world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
	world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{Shelter: gc.ShelterFull})

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 8, temp, "屋内はアンカー10へ半分寄せた 10+(5-10)/2 = 8 で屋外の 5 より暖かい")
}

func TestAmbientTemperatureAt_熱源は環境気温を押し上げる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)

	fire := world.ECS.NewEntity()
	world.Components.GridElement.Add(fire, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
	world.Components.HeatSource.Add(fire, &gc.HeatSource{Radius: 2, Warmth: 0.75})

	temp, err := query.AmbientTemperatureAt(world, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, -5, temp, "隣接の押し上げは 0.75*2/3*30=15℃ で -20 が -5 になる")
}

func TestAmbientTemperatureAt_半屋外タイルは世界温度を中間の強さで受ける(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)

	tile := world.ECS.NewEntity()
	world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
	world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{Shelter: gc.ShelterPartial})

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, -12, temp, "半屋外はアンカー10へ 3/4 寄せた 10+(-30)*3/4 = -12 になる")
}

func TestAmbientTemperatureAt_冬の屋内は屋外より暖かい(t *testing.T) {
	t.Parallel()
	const winterNoon consts.Turn = 24*1500 + 500

	outdoor := func() int {
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = winterNoon
		query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
		query.EnsureSeamlessBand(world)
		temp, err := query.AmbientTemperatureAt(world, 0, 0)
		require.NoError(t, err)
		return temp
	}()
	indoor := func() int {
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = winterNoon
		query.GetDungeon(world).CurrentStage = gc.NewDungeonStage(coldDungeonName, 1)
		tile := world.ECS.NewEntity()
		world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
		world.Components.TileEnvironment.Add(tile, &gc.TileEnvironment{Shelter: gc.ShelterFull})
		temp, err := query.AmbientTemperatureAt(world, 0, 0)
		require.NoError(t, err)
		return temp
	}()

	assert.Greater(t, indoor, outdoor, "冬でも屋内は屋外より暖かい退避先になる")
}

func TestAmbientTemperatureAt_StageFieldの基本気温を足す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 開始直後の春。世界温度 = 5。基本気温は StageField に確定済みの値を読む
	query.GetGameTime(world).TotalTurns = 0
	key := gc.NewDungeonStage(coldDungeonName, 1)
	query.GetDungeon(world).CurrentStage = key
	query.EnsureStageField(world, key).BaseTemp = 20

	temp, err := query.AmbientTemperatureAt(world, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 25, temp, "基本気温 20 に屋外世界温度 5 を足した 25 になる")
}

func TestCalculateEquippedInsulation(t *testing.T) {
	t.Parallel()

	t.Run("装備なしの場合は全て0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)

		insulation := query.CalculateEquippedInsulation(world, player)
		assert.Equal(t, 0, insulation.Cold)
		assert.Equal(t, 0, insulation.Heat)
	})

	t.Run("装備の断熱値が合算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)

		armor := world.ECS.NewEntity()
		world.Components.Wearable.Add(armor, &gc.Wearable{
			EquipmentCategory: gc.EquipmentTorso,
			InsulationCold:    10,
			InsulationHeat:    5,
		})
		world.Components.LocationEquipped.Add(armor, &gc.LocationEquipped{
			Owner: player,
		})

		helmet := world.ECS.NewEntity()
		world.Components.Wearable.Add(helmet, &gc.Wearable{
			EquipmentCategory: gc.EquipmentHead,
			InsulationCold:    3,
			InsulationHeat:    2,
		})
		world.Components.LocationEquipped.Add(helmet, &gc.LocationEquipped{
			Owner: player,
		})

		insulation := query.CalculateEquippedInsulation(world, player)
		assert.Equal(t, 13, insulation.Cold)
		assert.Equal(t, 7, insulation.Heat)
	})
}

func TestHeatSourceWarmthAt_距離に応じて減衰し半径外は無視する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	addHeatSource := func(x, y, radius consts.Tile, warmth float64) {
		e := world.ECS.NewEntity()
		world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		world.Components.HeatSource.Add(e, &gc.HeatSource{Radius: radius, Warmth: warmth})
	}
	addHeatSource(5, 5, 2, 0.6)
	addHeatSource(20, 20, 1, 9.9)

	assert.InDelta(t, 0.6, query.HeatSourceWarmthAt(world, 5, 5), 1e-9)
	assert.InDelta(t, 0.4, query.HeatSourceWarmthAt(world, 6, 5), 1e-9)
	assert.InDelta(t, 0.2, query.HeatSourceWarmthAt(world, 7, 5), 1e-9)
	assert.InDelta(t, 0.0, query.HeatSourceWarmthAt(world, 8, 5), 1e-9)
}

func TestHeatSourceWarmthAt_複数の熱源を加算する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	addHeatSource := func(x, y, radius consts.Tile, warmth float64) {
		e := world.ECS.NewEntity()
		world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		world.Components.HeatSource.Add(e, &gc.HeatSource{Radius: radius, Warmth: warmth})
	}
	addHeatSource(5, 5, 1, 0.5)
	addHeatSource(6, 6, 2, 0.3)

	assert.InDelta(t, 0.7, query.HeatSourceWarmthAt(world, 5, 5), 1e-9)
}
