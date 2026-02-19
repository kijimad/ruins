package dungeon

import (
	"testing"

	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectPlanner(t *testing.T) {
	t.Parallel()

	t.Run("空のプールの場合はエラーを返す", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			Name:        "テスト",
			PlannerPool: []PlannerWeight{},
		}
		_, err := SelectPlanner(def, 12345)
		assert.Error(t, err)
	})

	t.Run("単一要素のプールはその要素を返す", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
			},
			EnemyTableName: "洞窟敵",
			ItemTableName:  "洞窟アイテム",
		}
		result, err := SelectPlanner(def, 12345)
		require.NoError(t, err)
		assert.Equal(t, mapplanner.PlannerTypeCave.Name, result.Name)
		assert.Equal(t, "洞窟敵", result.EnemyTableName)
		assert.Equal(t, "洞窟アイテム", result.ItemTableName)
	})

	t.Run("重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeForest, Weight: 100},
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
			},
			EnemyTableName: "森",
			ItemTableName:  "森",
		}

		forestCount := 0
		caveCount := 0
		for i := range 1000 {
			result, err := SelectPlanner(def, uint64(i))
			require.NoError(t, err)
			switch result.Name {
			case mapplanner.PlannerTypeForest.Name:
				forestCount++
			case mapplanner.PlannerTypeCave.Name:
				caveCount++
			}
		}

		// 森の方が圧倒的に多いはず
		assert.Greater(t, forestCount, caveCount*10)
	})

	t.Run("重みが0のみの場合はエラーを返す", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			Name: "テスト",
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeRuins, Weight: 0},
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 0},
			},
		}
		_, err := SelectPlanner(def, 12345)
		assert.Error(t, err)
	})

	t.Run("同じシードでは同じ結果を返す", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeForest, Weight: 1},
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
				{PlannerType: mapplanner.PlannerTypeRuins, Weight: 1},
			},
		}

		result1, err := SelectPlanner(def, 12345)
		require.NoError(t, err)
		result2, err := SelectPlanner(def, 12345)
		require.NoError(t, err)
		assert.Equal(t, result1.Name, result2.Name)
	})

	t.Run("異なるシードでは異なる結果を返す可能性がある", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeForest, Weight: 1},
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
				{PlannerType: mapplanner.PlannerTypeRuins, Weight: 1},
			},
		}

		differentCount := 0
		for i := range 100 {
			result1, err := SelectPlanner(def, uint64(i))
			require.NoError(t, err)
			result2, err := SelectPlanner(def, uint64(i+1000))
			require.NoError(t, err)
			if result1.Name != result2.Name {
				differentCount++
			}
		}

		// 異なる結果が一定数あるはず
		assert.Greater(t, differentCount, 10)
	})
}
