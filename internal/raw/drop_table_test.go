package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDropTable_SelectByWeight_EmptyMaterial(t *testing.T) {
	t.Parallel()

	// 空文字列エントリを含むドロップテーブル
	dropTable := DropTable{
		Name: "テスト敵",
		Entries: []DropTableEntry{
			{Material: "", Weight: 0.5},     // 50%でドロップなし
			{Material: "アイテム", Weight: 0.5}, // 50%でアイテムドロップ
		},
	}

	// 複数回実行して両方のケースが発生することを確認
	emptyCount := 0
	itemCount := 0
	iterations := 1000

	rng := rand.New(rand.NewPCG(12345, 67890))
	for i := 0; i < iterations; i++ {
		result := dropTable.SelectByWeight(rng)
		switch result {
		case "":
			emptyCount++
		case "アイテム":
			itemCount++
		}
	}

	// 確率的に両方のケースが発生しているはず（厳密には50%ずつ）
	assert.Greater(t, emptyCount, 0, "空文字列が選択されるべき")
	assert.Greater(t, itemCount, 0, "アイテムが選択されるべき")

	// 大体50%ずつになっているはず（誤差を考慮して30-70%の範囲で確認）
	emptyRatio := float64(emptyCount) / float64(iterations)
	assert.Greater(t, emptyRatio, 0.3, "空文字列の確率が低すぎる")
	assert.Less(t, emptyRatio, 0.7, "空文字列の確率が高すぎる")
}

func TestDropTable_SelectByWeight_AllEmptyWeight(t *testing.T) {
	t.Parallel()

	// 重みが全て0のドロップテーブル
	dropTable := DropTable{
		Name: "テスト敵",
		Entries: []DropTableEntry{
			{Material: "アイテム1", Weight: 0},
			{Material: "アイテム2", Weight: 0},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := dropTable.SelectByWeight(rng)

	// 重みが全て0の場合は空文字列を返すべき
	assert.Equal(t, "", result, "重みが0の場合は空文字列を返すべき")
}

func TestDropTable_SelectByWeight_SingleEntry(t *testing.T) {
	t.Parallel()

	// エントリが1つだけのドロップテーブル
	dropTable := DropTable{
		Name: "テスト敵",
		Entries: []DropTableEntry{
			{Material: "確定アイテム", Weight: 1.0},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := dropTable.SelectByWeight(rng)

	assert.Equal(t, "確定アイテム", result, "エントリが1つの場合はそれが選択されるべき")
}

func TestDropTable_SelectByWeight_OnlyEmpty(t *testing.T) {
	t.Parallel()

	// 空文字列エントリのみのドロップテーブル
	dropTable := DropTable{
		Name: "テスト敵",
		Entries: []DropTableEntry{
			{Material: "", Weight: 1.0},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	result := dropTable.SelectByWeight(rng)

	assert.Equal(t, "", result, "空文字列エントリのみの場合は空文字列を返すべき")
}

func TestDropTable_SelectByWeight_MultipleEntries(t *testing.T) {
	t.Parallel()

	// 複数エントリのドロップテーブル
	dropTable := DropTable{
		Name: "テスト敵",
		Entries: []DropTableEntry{
			{Material: "レアアイテム", Weight: 0.1},  // 10%
			{Material: "コモンアイテム", Weight: 0.4}, // 40%
			{Material: "", Weight: 0.5},        // 50%でドロップなし
		},
	}

	// 各アイテムが選択されることを確認
	results := make(map[string]int)
	iterations := 10000

	rng := rand.New(rand.NewPCG(12345, 67890))
	for i := 0; i < iterations; i++ {
		result := dropTable.SelectByWeight(rng)
		results[result]++
	}

	// 全てのエントリが選択されているはず
	assert.Greater(t, results["レアアイテム"], 0, "レアアイテムが選択されるべき")
	assert.Greater(t, results["コモンアイテム"], 0, "コモンアイテムが選択されるべき")
	assert.Greater(t, results[""], 0, "空文字列が選択されるべき")

	// 大体期待確率になっているはず（誤差を考慮）
	rareRatio := float64(results["レアアイテム"]) / float64(iterations)
	commonRatio := float64(results["コモンアイテム"]) / float64(iterations)
	emptyRatio := float64(results[""]) / float64(iterations)

	assert.InDelta(t, 0.1, rareRatio, 0.05, "レアアイテムの確率が期待値から外れている")
	assert.InDelta(t, 0.4, commonRatio, 0.05, "コモンアイテムの確率が期待値から外れている")
	assert.InDelta(t, 0.5, emptyRatio, 0.05, "空文字列の確率が期待値から外れている")
}
