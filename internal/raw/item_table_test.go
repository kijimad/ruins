package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemTable_SelectByWeight_SingleEntry(t *testing.T) {
	t.Parallel()

	itemTable := ItemTable{
		Name: "テスト",
		Entries: []ItemTableEntry{
			{ItemName: "回復薬", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := itemTable.SelectByWeight(rng, 5)

	assert.Equal(t, "回復薬", result, "エントリが1つの場合はそれが選択されるべき")
}

func TestItemTable_SelectByWeight_MultipleEntries(t *testing.T) {
	t.Parallel()

	itemTable := ItemTable{
		Name: "通常",
		Entries: []ItemTableEntry{
			{ItemName: "回復薬", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{ItemName: "回復スプレー", Weight: 0.8, MinDepth: 1, MaxDepth: 20},
			{ItemName: "手榴弾", Weight: 0.5, MinDepth: 1, MaxDepth: 20},
		},
	}

	// 各アイテムが選択されることを確認
	results := make(map[string]int)
	iterations := 10000

	rng := rand.New(rand.NewPCG(12345, 67890))
	for i := 0; i < iterations; i++ {
		result := itemTable.SelectByWeight(rng, 5)
		results[result]++
	}

	// 全てのアイテムが選択されているはず
	assert.Greater(t, results["回復薬"], 0, "回復薬が選択されるべき")
	assert.Greater(t, results["回復スプレー"], 0, "回復スプレーが選択されるべき")
	assert.Greater(t, results["手榴弾"], 0, "手榴弾が選択されるべき")

	// 重みに応じた確率になっているはず
	totalWeight := 1.0 + 0.8 + 0.5
	expectedRatio1 := 1.0 / totalWeight
	expectedRatio2 := 0.8 / totalWeight
	expectedRatio3 := 0.5 / totalWeight

	ratio1 := float64(results["回復薬"]) / float64(iterations)
	ratio2 := float64(results["回復スプレー"]) / float64(iterations)
	ratio3 := float64(results["手榴弾"]) / float64(iterations)

	assert.InDelta(t, expectedRatio1, ratio1, 0.05, "回復薬の確率が期待値から外れている")
	assert.InDelta(t, expectedRatio2, ratio2, 0.05, "回復スプレーの確率が期待値から外れている")
	assert.InDelta(t, expectedRatio3, ratio3, 0.05, "手榴弾の確率が期待値から外れている")
}

func TestItemTable_SelectByWeight_AllZeroWeight(t *testing.T) {
	t.Parallel()

	itemTable := ItemTable{
		Name: "テスト",
		Entries: []ItemTableEntry{
			{ItemName: "アイテム1", Weight: 0, MinDepth: 1, MaxDepth: 10},
			{ItemName: "アイテム2", Weight: 0, MinDepth: 1, MaxDepth: 10},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := itemTable.SelectByWeight(rng, 5)

	assert.Equal(t, "", result, "重みが全て0の場合は空文字列を返すべき")
}

func TestItemTable_SelectByWeight_EmptyEntries(t *testing.T) {
	t.Parallel()

	itemTable := ItemTable{
		Name:    "空",
		Entries: []ItemTableEntry{},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := itemTable.SelectByWeight(rng, 1)

	assert.Equal(t, "", result, "エントリが空の場合は空文字列を返すべき")
}

func TestItemTable_SelectByWeight_Reproducibility(t *testing.T) {
	t.Parallel()

	itemTable := ItemTable{
		Name: "通常",
		Entries: []ItemTableEntry{
			{ItemName: "アイテムA", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{ItemName: "アイテムB", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{ItemName: "アイテムC", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}

	// 同じシードで複数回実行して同じ結果になることを確認
	seed := uint64(99999)
	rng1 := rand.New(rand.NewPCG(seed, seed+1))
	rng2 := rand.New(rand.NewPCG(seed, seed+1))

	for i := 0; i < 100; i++ {
		result1 := itemTable.SelectByWeight(rng1, 5)
		result2 := itemTable.SelectByWeight(rng2, 5)
		assert.Equal(t, result1, result2, "同じシードで同じ結果が得られるべき")
	}
}
