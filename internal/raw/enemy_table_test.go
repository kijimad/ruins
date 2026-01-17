package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnemyTable_SelectByWeight_SingleEntry(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "テスト",
		Entries: []EnemyTableEntry{
			{EnemyName: "スライム", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := enemyTable.SelectByWeight(rng, 5)

	assert.Equal(t, "スライム", result, "エントリが1つの場合はそれが選択されるべき")
}

func TestEnemyTable_SelectByWeight_MultipleEntries(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "通常",
		Entries: []EnemyTableEntry{
			{EnemyName: "スライム", Weight: 1.2, MinDepth: 1, MaxDepth: 20},
			{EnemyName: "火の玉", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{EnemyName: "軽戦車", Weight: 0.8, MinDepth: 1, MaxDepth: 20},
		},
	}

	// 各敵が選択されることを確認
	results := make(map[string]int)
	iterations := 10000

	rng := rand.New(rand.NewPCG(12345, 67890))
	for i := 0; i < iterations; i++ {
		result := enemyTable.SelectByWeight(rng, 5)
		results[result]++
	}

	// 全ての敵が選択されているはず
	assert.Greater(t, results["スライム"], 0, "スライムが選択されるべき")
	assert.Greater(t, results["火の玉"], 0, "火の玉が選択されるべき")
	assert.Greater(t, results["軽戦車"], 0, "軽戦車が選択されるべき")

	// 重みに応じた確率になっているはず
	totalWeight := 1.2 + 1.0 + 0.8
	expectedRatio1 := 1.2 / totalWeight
	expectedRatio2 := 1.0 / totalWeight
	expectedRatio3 := 0.8 / totalWeight

	ratio1 := float64(results["スライム"]) / float64(iterations)
	ratio2 := float64(results["火の玉"]) / float64(iterations)
	ratio3 := float64(results["軽戦車"]) / float64(iterations)

	assert.InDelta(t, expectedRatio1, ratio1, 0.05, "スライムの確率が期待値から外れている")
	assert.InDelta(t, expectedRatio2, ratio2, 0.05, "火の玉の確率が期待値から外れている")
	assert.InDelta(t, expectedRatio3, ratio3, 0.05, "軽戦車の確率が期待値から外れている")
}

func TestEnemyTable_SelectByWeight_AllZeroWeight(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "テスト",
		Entries: []EnemyTableEntry{
			{EnemyName: "敵1", Weight: 0, MinDepth: 1, MaxDepth: 10},
			{EnemyName: "敵2", Weight: 0, MinDepth: 1, MaxDepth: 10},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := enemyTable.SelectByWeight(rng, 5)

	assert.Equal(t, "", result, "重みが全て0の場合は空文字列を返すべき")
}

func TestEnemyTable_SelectByWeight_EmptyEntries(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name:    "空",
		Entries: []EnemyTableEntry{},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := enemyTable.SelectByWeight(rng, 1)

	assert.Equal(t, "", result, "エントリが空の場合は空文字列を返すべき")
}

func TestEnemyTable_SelectByWeight_Reproducibility(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "通常",
		Entries: []EnemyTableEntry{
			{EnemyName: "敵A", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{EnemyName: "敵B", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{EnemyName: "敵C", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}

	// 同じシードで複数回実行して同じ結果になることを確認
	seed := uint64(99999)
	rng1 := rand.New(rand.NewPCG(seed, seed+1))
	rng2 := rand.New(rand.NewPCG(seed, seed+1))

	for i := 0; i < 100; i++ {
		result1 := enemyTable.SelectByWeight(rng1, 5)
		result2 := enemyTable.SelectByWeight(rng2, 5)
		assert.Equal(t, result1, result2, "同じシードで同じ結果が得られるべき")
	}
}

func TestEnemyTable_SelectByWeight_DepthFiltering_MinDepth(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "深度テスト",
		Entries: []EnemyTableEntry{
			{EnemyName: "弱い敵", Weight: 1.0, MinDepth: 1, MaxDepth: 5},
			{EnemyName: "中級の敵", Weight: 1.0, MinDepth: 5, MaxDepth: 10},
			{EnemyName: "強い敵", Weight: 1.0, MinDepth: 10, MaxDepth: 20},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// 深度1: 弱い敵のみ選択可能
	results := make(map[string]int)
	for i := 0; i < 1000; i++ {
		result := enemyTable.SelectByWeight(rng, 1)
		if result != "" {
			results[result]++
		}
	}
	assert.Greater(t, results["弱い敵"], 0, "深度1では弱い敵が選択されるべき")
	assert.Equal(t, 0, results["中級の敵"], "深度1では中級の敵は選択されない")
	assert.Equal(t, 0, results["強い敵"], "深度1では強い敵は選択されない")

	// 深度5: 弱い敵と中級の敵が選択可能
	results = make(map[string]int)
	for i := 0; i < 1000; i++ {
		result := enemyTable.SelectByWeight(rng, 5)
		if result != "" {
			results[result]++
		}
	}
	assert.Greater(t, results["弱い敵"], 0, "深度5では弱い敵が選択されるべき")
	assert.Greater(t, results["中級の敵"], 0, "深度5では中級の敵が選択されるべき")
	assert.Equal(t, 0, results["強い敵"], "深度5では強い敵は選択されない")

	// 深度15: 強い敵のみ選択可能
	results = make(map[string]int)
	for i := 0; i < 1000; i++ {
		result := enemyTable.SelectByWeight(rng, 15)
		if result != "" {
			results[result]++
		}
	}
	assert.Equal(t, 0, results["弱い敵"], "深度15では弱い敵は選択されない")
	assert.Equal(t, 0, results["中級の敵"], "深度15では中級の敵は選択されない")
	assert.Greater(t, results["強い敵"], 0, "深度15では強い敵が選択されるべき")
}

func TestEnemyTable_SelectByWeight_DepthFiltering_NoMatch(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "深度範囲外",
		Entries: []EnemyTableEntry{
			{EnemyName: "敵1", Weight: 1.0, MinDepth: 10, MaxDepth: 20},
			{EnemyName: "敵2", Weight: 1.0, MinDepth: 20, MaxDepth: 30},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// 深度5: 全ての敵が範囲外
	result := enemyTable.SelectByWeight(rng, 5)
	assert.Equal(t, "", result, "深度範囲外の場合は空文字列を返すべき")

	// 深度50: 全ての敵が範囲外
	result = enemyTable.SelectByWeight(rng, 50)
	assert.Equal(t, "", result, "深度範囲外の場合は空文字列を返すべき")
}

func TestEnemyTable_SelectByWeight_DepthFiltering_NoRestriction(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "制限なし",
		Entries: []EnemyTableEntry{
			{EnemyName: "常に出現", Weight: 1.0, MinDepth: 0, MaxDepth: 0},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// MinDepth=0, MaxDepth=0は深度制限なし
	result1 := enemyTable.SelectByWeight(rng, 1)
	assert.Equal(t, "常に出現", result1, "深度1で選択可能")

	result100 := enemyTable.SelectByWeight(rng, 100)
	assert.Equal(t, "常に出現", result100, "深度100でも選択可能")

	result1000 := enemyTable.SelectByWeight(rng, 1000)
	assert.Equal(t, "常に出現", result1000, "深度1000でも選択可能")
}

func TestEnemyTable_SelectByWeight_DepthFiltering_MaxDepthOnly(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "最大深度のみ",
		Entries: []EnemyTableEntry{
			{EnemyName: "序盤限定", Weight: 1.0, MinDepth: 0, MaxDepth: 10},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// MaxDepthのみ設定 (MinDepth=0は制限なし)
	result1 := enemyTable.SelectByWeight(rng, 1)
	assert.Equal(t, "序盤限定", result1, "深度1で選択可能")

	result10 := enemyTable.SelectByWeight(rng, 10)
	assert.Equal(t, "序盤限定", result10, "深度10で選択可能")

	result11 := enemyTable.SelectByWeight(rng, 11)
	assert.Equal(t, "", result11, "深度11では選択不可")
}

func TestEnemyTable_SelectByWeight_DepthFiltering_MinDepthOnly(t *testing.T) {
	t.Parallel()

	enemyTable := EnemyTable{
		Name: "最小深度のみ",
		Entries: []EnemyTableEntry{
			{EnemyName: "後半限定", Weight: 1.0, MinDepth: 20, MaxDepth: 0},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// MinDepthのみ設定 (MaxDepth=0は制限なし)
	result10 := enemyTable.SelectByWeight(rng, 10)
	assert.Equal(t, "", result10, "深度10では選択不可")

	result20 := enemyTable.SelectByWeight(rng, 20)
	assert.Equal(t, "後半限定", result20, "深度20で選択可能")

	result100 := enemyTable.SelectByWeight(rng, 100)
	assert.Equal(t, "後半限定", result100, "深度100でも選択可能")
}
