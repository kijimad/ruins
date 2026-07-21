package dungeon

import (
	"testing"

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
		kind, found := GetStageKind("亡者の森")
		require.True(t, found)
		assert.Equal(t, "亡者の森", kind.Name())
		d, ok := kind.(*DungeonKind)
		require.True(t, ok, "通常ダンジョンは DungeonKind")
		assert.Equal(t, 20, d.TotalFloors())
	})

	t.Run("オーバーワールドは OverworldKind として引ける", func(t *testing.T) {
		t.Parallel()
		kind, found := GetStageKind("オーバーワールド")
		require.True(t, found)
		_, ok := kind.(*OverworldKind)
		assert.True(t, ok, "オーバーワールドは OverworldKind でフロアを生成しない別の型")
	})

	t.Run("存在しないダンジョンはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, found := GetStageKind("存在しないダンジョン")
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
		assert.NotEmpty(t, DungeonForest.PlannerPool())
	})

	t.Run("DungeonCaveの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "灰の洞窟", DungeonCave.Name())
		assert.Equal(t, 20, DungeonCave.TotalFloors())
		assert.Equal(t, "洞窟", DungeonCave.EnemyTableName())
		assert.Equal(t, "洞窟", DungeonCave.ItemTableName())
		assert.NotEmpty(t, DungeonCave.PlannerPool())
	})

	t.Run("DungeonRuinsの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "忘却の廃都", DungeonRuins.Name())
		assert.Equal(t, 20, DungeonRuins.TotalFloors())
		assert.Equal(t, "廃墟", DungeonRuins.EnemyTableName())
		assert.Equal(t, "廃墟", DungeonRuins.ItemTableName())
		assert.NotEmpty(t, DungeonRuins.PlannerPool())
	})
}
