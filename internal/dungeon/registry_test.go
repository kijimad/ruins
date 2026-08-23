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
		assert.NotEmpty(t, d.PlannerPool())
	}
}

func TestGetDungeonByName(t *testing.T) {
	t.Parallel()

	t.Run("存在するダンジョンを取得できる", func(t *testing.T) {
		t.Parallel()
		def, found := GetStageDefinition("Dead forest")
		require.True(t, found)
		assert.Equal(t, "Dead forest", def.Name())
		_, ok := def.(*DungeonDefinition)
		require.True(t, ok, "通常ダンジョンは DungeonDefinition")
	})

	t.Run("オーバーワールドは OverworldDefinition として引ける", func(t *testing.T) {
		t.Parallel()
		def, found := GetStageDefinition("Overworld")
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
		assert.Equal(t, "Dead forest", DungeonForest.Name())
		assert.Equal(t, "forest", DungeonForest.EnemyTableName())
		assert.Equal(t, "forest", DungeonForest.ItemTableName())
		assert.Equal(t, "Hunters once ventured into this frozen forest.\nFew returned. The cold reaches the bone.", DungeonForest.Description())
		assert.Equal(t, "forest1", DungeonForest.ImageKey())
		assert.Equal(t, 0, DungeonForest.BaseTemperature())
		assert.NotEmpty(t, DungeonForest.PlannerPool())
	})

	t.Run("DungeonCaveの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Ash cave", DungeonCave.Name())
		assert.Equal(t, "cave", DungeonCave.EnemyTableName())
		assert.Equal(t, "cave", DungeonCave.ItemTableName())
		assert.Equal(t, "Frost crystals run like veins through the gray rock.\nThe deeper you go, the quieter it grows.", DungeonCave.Description())
		assert.Equal(t, "cave1", DungeonCave.ImageKey())
		assert.Equal(t, 5, DungeonCave.BaseTemperature())
		assert.NotEmpty(t, DungeonCave.PlannerPool())
	})

	t.Run("DungeonRuinsの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Forgotten ruins", DungeonRuins.Name())
		assert.Equal(t, "ruins_area", DungeonRuins.EnemyTableName())
		assert.Equal(t, "ruins_area", DungeonRuins.ItemTableName())
		assert.Equal(t, "An ancient city stands frozen in place.\nWho forgot what, no one remembers now.", DungeonRuins.Description())
		assert.Equal(t, "city1", DungeonRuins.ImageKey())
		assert.Equal(t, 15, DungeonRuins.BaseTemperature())
		assert.NotEmpty(t, DungeonRuins.PlannerPool())
	})

	t.Run("DungeonOverworldの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Overworld", DungeonOverworld.Name())
		assert.Equal(t, 0, DungeonOverworld.BaseTemperature())
		chunkW, chunkH, cols, rows := DungeonOverworld.BandShape()
		assert.Equal(t, consts.Tile(24), chunkW)
		assert.Equal(t, consts.Tile(24), chunkH)
		assert.Equal(t, consts.Chunk(7), cols)
		assert.Equal(t, consts.Chunk(9), rows)
	})
}
