package dungeon

import (
	"math/rand/v2"
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
		rng := rand.New(rand.NewPCG(12345, 0))
		_, err := SelectPlanner(def, rng)
		assert.Error(t, err)
	})

	t.Run("単一要素のプールはその要素を返す", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
			},
		}
		rng := rand.New(rand.NewPCG(12345, 0))
		result, err := SelectPlanner(def, rng)
		require.NoError(t, err)
		assert.Equal(t, mapplanner.PlannerTypeCave.Name, result.Name)
	})

	t.Run("重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeForest, Weight: 100},
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
			},
		}

		forestCount := 0
		caveCount := 0
		rng := rand.New(rand.NewPCG(12345, 0))
		for range 100 {
			result, err := SelectPlanner(def, rng)
			require.NoError(t, err)
			switch result.Name {
			case mapplanner.PlannerTypeForest.Name:
				forestCount++
			case mapplanner.PlannerTypeCave.Name:
				caveCount++
			}
		}

		// 森の方が多いはず
		assert.Greater(t, forestCount, caveCount)
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
		rng := rand.New(rand.NewPCG(12345, 0))
		_, err := SelectPlanner(def, rng)
		assert.Error(t, err)
	})

	t.Run("同じシードで生成された初期状態のRNGでは同じ結果を返す", func(t *testing.T) {
		t.Parallel()
		def := Definition{
			PlannerPool: []PlannerWeight{
				{PlannerType: mapplanner.PlannerTypeForest, Weight: 1},
				{PlannerType: mapplanner.PlannerTypeCave, Weight: 1},
				{PlannerType: mapplanner.PlannerTypeRuins, Weight: 1},
			},
		}

		rng1 := rand.New(rand.NewPCG(12345, 0))
		result1, err := SelectPlanner(def, rng1)
		require.NoError(t, err)

		rng2 := rand.New(rand.NewPCG(12345, 0))
		result2, err := SelectPlanner(def, rng2)
		require.NoError(t, err)

		assert.Equal(t, result1.Name, result2.Name)
	})

	t.Run("異なるシードのRNGでは異なる結果を返す可能性がある", func(t *testing.T) {
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
			rng1 := rand.New(rand.NewPCG(uint64(i), 0))
			result1, err := SelectPlanner(def, rng1)
			require.NoError(t, err)

			rng2 := rand.New(rand.NewPCG(uint64(i+1000), 0))
			result2, err := SelectPlanner(def, rng2)
			require.NoError(t, err)

			if result1.Name != result2.Name {
				differentCount++
			}
		}

		// 異なる結果が一定数あるはず
		assert.Greater(t, differentCount, 10)
	})
}
