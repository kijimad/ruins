package dungeon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAllDungeons(t *testing.T) {
	t.Parallel()

	dungeons := GetAllDungeons()

	assert.NotEmpty(t, dungeons)
	assert.GreaterOrEqual(t, len(dungeons), 3)

	// 全てのダンジョンが有効な設定を持っている
	for _, d := range dungeons {
		assert.NotEmpty(t, d.Name)
		assert.Greater(t, d.TotalFloors, 0)
		assert.NotEmpty(t, d.PlannerPool)
	}
}

func TestGetDungeonByName(t *testing.T) {
	t.Parallel()

	t.Run("存在するダンジョンを取得できる", func(t *testing.T) {
		t.Parallel()
		d, found := GetDungeon("亡者の森")
		assert.True(t, found)
		assert.Equal(t, "亡者の森", d.Name)
		assert.Equal(t, 10, d.TotalFloors)
	})

	t.Run("存在しないダンジョンはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, found := GetDungeon("存在しないダンジョン")
		assert.False(t, found)
	})
}

func TestDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("DungeonForestの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "亡者の森", DungeonForest.Name)
		assert.Equal(t, 10, DungeonForest.TotalFloors)
		assert.Equal(t, "森", DungeonForest.EnemyTableName)
		assert.Equal(t, "森", DungeonForest.ItemTableName)
		assert.NotEmpty(t, DungeonForest.PlannerPool)
	})

	t.Run("DungeonCaveの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "灰の洞窟", DungeonCave.Name)
		assert.Equal(t, 15, DungeonCave.TotalFloors)
		assert.Equal(t, "洞窟", DungeonCave.EnemyTableName)
		assert.Equal(t, "洞窟", DungeonCave.ItemTableName)
		assert.NotEmpty(t, DungeonCave.PlannerPool)
	})

	t.Run("DungeonRuinsの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "忘却の廃都", DungeonRuins.Name)
		assert.Equal(t, 20, DungeonRuins.TotalFloors)
		assert.Equal(t, "廃墟", DungeonRuins.EnemyTableName)
		assert.Equal(t, "廃墟", DungeonRuins.ItemTableName)
		assert.NotEmpty(t, DungeonRuins.PlannerPool)
	})
}
