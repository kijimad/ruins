package dungeon

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllDungeons(t *testing.T) {
	t.Parallel()

	dungeons := GetAllDungeons()

	assert.NotEmpty(t, dungeons)
	assert.GreaterOrEqual(t, len(dungeons), 3)

	// 全てのダンジョンが有効な設定を持っている
	for _, d := range dungeons {
		assert.NotEmpty(t, d.Name())
		assert.Positive(t, d.TotalFloors())
		assert.NotEmpty(t, d.PlannerPool())
	}
}

func TestGetAllDungeonNames(t *testing.T) {
	t.Parallel()

	names := GetAllDungeonNames()
	assert.NotEmpty(t, names)
	assert.Len(t, names, len(GetAllDungeons()))

	for _, name := range names {
		assert.NotEmpty(t, name)
	}
}

func TestGetDungeonByName(t *testing.T) {
	t.Parallel()

	t.Run("存在するダンジョンを取得できる", func(t *testing.T) {
		t.Parallel()
		def, found := GetStageDefinition("亡者の森")
		require.True(t, found)
		assert.Equal(t, "亡者の森", def.Name())
		d, ok := def.(*DungeonDefinition)
		require.True(t, ok, "通常ダンジョンは DungeonDefinition")
		assert.Equal(t, 20, d.TotalFloors())
	})

	t.Run("オーバーワールドは OverworldDefinition として引ける", func(t *testing.T) {
		t.Parallel()
		def, found := GetStageDefinition("オーバーワールド")
		require.True(t, found)
		_, ok := def.(*OverworldDefinition)
		assert.True(t, ok, "オーバーワールドは OverworldDefinition でフロアを生成しない別の型")
	})

	t.Run("存在しないダンジョンはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, found := GetStageDefinition("存在しないダンジョン")
		assert.False(t, found)
	})
}

func TestDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("DungeonForestの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "亡者の森", DungeonForest.Name())
		assert.Equal(t, 20, DungeonForest.TotalFloors())
		assert.Equal(t, "森", DungeonForest.EnemyTableName())
		assert.Equal(t, "森", DungeonForest.ItemTableName())
		assert.Equal(t, "凍りついた森に、かつて猟師たちが分け入った。\n戻った者は少ない。冷気が骨まで届く。", DungeonForest.Description())
		assert.Equal(t, "forest1", DungeonForest.ImageKey())
		assert.Equal(t, 0, DungeonForest.BaseTemperature())
		assert.NotEmpty(t, DungeonForest.PlannerPool())
	})

	t.Run("DungeonCaveの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "灰の洞窟", DungeonCave.Name())
		assert.Equal(t, 20, DungeonCave.TotalFloors())
		assert.Equal(t, "洞窟", DungeonCave.EnemyTableName())
		assert.Equal(t, "洞窟", DungeonCave.ItemTableName())
		assert.Equal(t, "灰色の岩壁に凍晶が脈のように走っている。\n奥に進むほど、静かになる。", DungeonCave.Description())
		assert.Equal(t, "cave1", DungeonCave.ImageKey())
		assert.Equal(t, 5, DungeonCave.BaseTemperature())
		assert.NotEmpty(t, DungeonCave.PlannerPool())
	})

	t.Run("DungeonRuinsの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "忘却の廃都", DungeonRuins.Name())
		assert.Equal(t, 20, DungeonRuins.TotalFloors())
		assert.Equal(t, "廃墟", DungeonRuins.EnemyTableName())
		assert.Equal(t, "廃墟", DungeonRuins.ItemTableName())
		assert.Equal(t, "古代の都市が、そのまま凍りついている。\n誰が何を忘れたのか、もう誰も知らない。", DungeonRuins.Description())
		assert.Equal(t, "city1", DungeonRuins.ImageKey())
		assert.Equal(t, 15, DungeonRuins.BaseTemperature())
		assert.NotEmpty(t, DungeonRuins.PlannerPool())
	})

	t.Run("DungeonOverworldの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "オーバーワールド", DungeonOverworld.Name())
		assert.Equal(t, 0, DungeonOverworld.BaseTemperature())
		chunkW, chunkH, cols, rows := DungeonOverworld.BandShape()
		assert.Equal(t, consts.Tile(24), chunkW)
		assert.Equal(t, consts.Tile(24), chunkH)
		assert.Equal(t, consts.Chunk(7), cols)
		assert.Equal(t, consts.Chunk(9), rows)
	})
}
