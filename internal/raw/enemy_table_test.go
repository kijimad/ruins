package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnemyTable_SelectByWeight_SingleEntry(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name: "テスト",
		Entries: []oapi.EnemyTableEntry{
			{Id: "スライム", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result, err := SelectEnemyByWeight(enemyTable, rng, 5)
	require.NoError(t, err)

	assert.Equal(t, "スライム", result, "エントリが1つの場合はそれが選択されるべき")
}

func TestSelectEnemyByWeight_危険度が最小未満はエラー(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name:    "テスト",
		Entries: []oapi.EnemyTableEntry{{Id: "スライム", Weight: 1.0, MinDanger: 1, MaxDanger: 20}},
	}
	rng := rand.New(rand.NewPCG(1, 2))
	_, err := SelectEnemyByWeight(enemyTable, rng, MinDanger-1)
	require.ErrorContains(t, err, "危険度")
}

func TestEnemyTable_SelectByWeight_MultipleEntries(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name: "通常",
		Entries: []oapi.EnemyTableEntry{
			{Id: "スライム", Weight: 1.2, MinDanger: 1, MaxDanger: 20},
			{Id: "火の玉", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
			{Id: "軽戦車", Weight: 0.8, MinDanger: 1, MaxDanger: 20},
		},
	}

	// 各敵が選択されることを確認
	results := make(map[string]int)
	iterations := 10000

	rng := rand.New(rand.NewPCG(12345, 67890))
	for range iterations {
		result, err := SelectEnemyByWeight(enemyTable, rng, 5)
		require.NoError(t, err)
		results[result]++
	}

	// 全ての敵が選択されているはず
	assert.Positive(t, results["スライム"], "スライムが選択されるべき")
	assert.Positive(t, results["火の玉"], "火の玉が選択されるべき")
	assert.Positive(t, results["軽戦車"], "軽戦車が選択されるべき")

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

	enemyTable := oapi.EnemyTable{
		Name: "テスト",
		Entries: []oapi.EnemyTableEntry{
			{Id: "敵1", Weight: 0, MinDanger: 1, MaxDanger: 10},
			{Id: "敵2", Weight: 0, MinDanger: 1, MaxDanger: 10},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result, err := SelectEnemyByWeight(enemyTable, rng, 5)
	require.NoError(t, err)

	assert.Empty(t, result, "重みが全て0の場合は空文字列を返すべき")
}

func TestEnemyTable_SelectByWeight_EmptyEntries(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name:    "空",
		Entries: []oapi.EnemyTableEntry{},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result, err := SelectEnemyByWeight(enemyTable, rng, 1)
	require.NoError(t, err)

	assert.Empty(t, result, "エントリが空の場合は空文字列を返すべき")
}

func TestEnemyTable_SelectByWeight_Reproducibility(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name: "通常",
		Entries: []oapi.EnemyTableEntry{
			{Id: "敵A", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
			{Id: "敵B", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
			{Id: "敵C", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
		},
	}

	// 同じシードで複数回実行して同じ結果になることを確認
	seed := uint64(99999)
	rng1 := rand.New(rand.NewPCG(seed, seed+1))
	rng2 := rand.New(rand.NewPCG(seed, seed+1))

	for range 100 {
		result1, err1 := SelectEnemyByWeight(enemyTable, rng1, 5)
		result2, err2 := SelectEnemyByWeight(enemyTable, rng2, 5)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, result1, result2, "同じシードで同じ結果が得られるべき")
	}
}

func TestEnemyTable_SelectByWeight_DangerFiltering_MinDanger(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name: "危険度テスト",
		Entries: []oapi.EnemyTableEntry{
			{Id: "弱い敵", Weight: 1.0, MinDanger: 1, MaxDanger: 5},
			{Id: "中級の敵", Weight: 1.0, MinDanger: 5, MaxDanger: 10},
			{Id: "強い敵", Weight: 1.0, MinDanger: 10, MaxDanger: 20},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// 危険度1: 弱い敵のみ選択可能
	results := make(map[string]int)
	for range 1000 {
		result, err := SelectEnemyByWeight(enemyTable, rng, 1)
		require.NoError(t, err)
		if result != "" {
			results[result]++
		}
	}
	assert.Positive(t, results["弱い敵"], "危険度1では弱い敵が選択されるべき")
	assert.Equal(t, 0, results["中級の敵"], "危険度1では中級の敵は選択されない")
	assert.Equal(t, 0, results["強い敵"], "危険度1では強い敵は選択されない")

	// 危険度5: 弱い敵と中級の敵が選択可能
	results = make(map[string]int)
	for range 1000 {
		result, err := SelectEnemyByWeight(enemyTable, rng, 5)
		require.NoError(t, err)
		if result != "" {
			results[result]++
		}
	}
	assert.Positive(t, results["弱い敵"], "危険度5では弱い敵が選択されるべき")
	assert.Positive(t, results["中級の敵"], "危険度5では中級の敵が選択されるべき")
	assert.Equal(t, 0, results["強い敵"], "危険度5では強い敵は選択されない")

	// 危険度15: 強い敵のみ選択可能
	results = make(map[string]int)
	for range 1000 {
		result, err := SelectEnemyByWeight(enemyTable, rng, 15)
		require.NoError(t, err)
		if result != "" {
			results[result]++
		}
	}
	assert.Equal(t, 0, results["弱い敵"], "危険度15では弱い敵は選択されない")
	assert.Equal(t, 0, results["中級の敵"], "危険度15では中級の敵は選択されない")
	assert.Positive(t, results["強い敵"], "危険度15では強い敵が選択されるべき")
}

func TestEnemyTable_SelectByWeight_DangerFiltering_NoMatch(t *testing.T) {
	t.Parallel()

	enemyTable := oapi.EnemyTable{
		Name: "危険度範囲外",
		Entries: []oapi.EnemyTableEntry{
			{Id: "敵1", Weight: 1.0, MinDanger: 10, MaxDanger: 20},
			{Id: "敵2", Weight: 1.0, MinDanger: 20, MaxDanger: 30},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))

	// 危険度5: 全ての敵が範囲外
	result, err := SelectEnemyByWeight(enemyTable, rng, 5)
	require.NoError(t, err)
	assert.Empty(t, result, "危険度範囲外の場合は空文字列を返すべき")

	// 危険度50: 全ての敵が範囲外
	result, err = SelectEnemyByWeight(enemyTable, rng, 50)
	require.NoError(t, err)
	assert.Empty(t, result, "危険度範囲外の場合は空文字列を返すべき")
}
