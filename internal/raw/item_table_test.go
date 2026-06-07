package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のMasterを作成する。アイテムグループとアイテムテーブルを含む
func newTestMasterForItemTable(groups []oapi.ItemGroup, table oapi.ItemTable) *Master {
	groupIndex := make(map[string]int)
	for i, g := range groups {
		groupIndex[g.Name] = i
	}
	return &Master{
		Raws: Raws{
			ItemGroups: groups,
			ItemTables: []oapi.ItemTable{table},
		},
		ItemGroupIndex: groupIndex,
		ItemTableIndex: map[string]int{table.Name: 0},
	}
}

var testGroups = []oapi.ItemGroup{
	{
		Name:    "回復",
		Subtype: oapi.Distribution,
		Entries: []oapi.ItemGroupEntry{
			{ItemName: "回復薬", Weight: 1.0, PackMin: 1, PackMax: 1},
		},
	},
	{
		Name:    "武器",
		Subtype: oapi.Distribution,
		Entries: []oapi.ItemGroupEntry{
			{ItemName: "毒消し", Weight: 0.8, PackMin: 1, PackMax: 1},
			{ItemName: "手榴弾", Weight: 0.5, PackMin: 1, PackMax: 1},
		},
	},
	{
		Name:    "素材",
		Subtype: oapi.Distribution,
		Entries: []oapi.ItemGroupEntry{
			{ItemName: "アイテム1", Weight: 1.0, PackMin: 1, PackMax: 1},
			{ItemName: "アイテム2", Weight: 1.0, PackMin: 1, PackMax: 1},
		},
	},
}

func TestItemTable_SelectByWeight_SingleEntry(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name: "テスト",
		Entries: []oapi.ItemTableEntry{
			{GroupName: "回復", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}
	master := newTestMasterForItemTable(testGroups, table)

	rng := rand.New(rand.NewPCG(12345, 67890))
	result, err := SelectItemByWeight(master, table, rng, 5)
	require.NoError(t, err)

	assert.Equal(t, "回復薬", result, "グループに1アイテムのみの場合はそれが選択されるべき")
}

func TestItemTable_SelectByWeight_MultipleEntries(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name: "通常",
		Entries: []oapi.ItemTableEntry{
			{GroupName: "回復", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{GroupName: "武器", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}
	master := newTestMasterForItemTable(testGroups, table)

	results := make(map[string]int)
	iterations := 10000

	rng := rand.New(rand.NewPCG(12345, 67890))
	for i := 0; i < iterations; i++ {
		result, err := SelectItemByWeight(master, table, rng, 5)
		require.NoError(t, err)
		results[result]++
	}

	// グループ内の全アイテムが選択されることを確認
	assert.Greater(t, results["回復薬"], 0, "回復薬が選択されるべき")
	assert.Greater(t, results["毒消し"], 0, "毒消しが選択されるべき")
	assert.Greater(t, results["手榴弾"], 0, "手榴弾が選択されるべき")
}

func TestItemTable_SelectByWeight_AllZeroWeight(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name: "テスト",
		Entries: []oapi.ItemTableEntry{
			{GroupName: "回復", Weight: 0, MinDepth: 1, MaxDepth: 10},
			{GroupName: "武器", Weight: 0, MinDepth: 1, MaxDepth: 10},
		},
	}
	master := newTestMasterForItemTable(testGroups, table)

	rng := rand.New(rand.NewPCG(12345, 67890))
	result, err := SelectItemByWeight(master, table, rng, 5)
	require.NoError(t, err)

	assert.Equal(t, "", result, "重みが全て0の場合は空文字列を返すべき")
}

func TestItemTable_SelectByWeight_EmptyEntries(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name:    "空",
		Entries: []oapi.ItemTableEntry{},
	}
	master := newTestMasterForItemTable(testGroups, table)

	rng := rand.New(rand.NewPCG(12345, 67890))
	result, err := SelectItemByWeight(master, table, rng, 1)
	require.NoError(t, err)

	assert.Equal(t, "", result, "エントリが空の場合は空文字列を返すべき")
}

func TestItemTable_SelectByWeight_Reproducibility(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name: "通常",
		Entries: []oapi.ItemTableEntry{
			{GroupName: "回復", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{GroupName: "武器", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
			{GroupName: "素材", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
		},
	}
	master := newTestMasterForItemTable(testGroups, table)

	seed := uint64(99999)
	rng1 := rand.New(rand.NewPCG(seed, seed+1))
	rng2 := rand.New(rand.NewPCG(seed, seed+1))

	for i := 0; i < 100; i++ {
		result1, err1 := SelectItemByWeight(master, table, rng1, 5)
		result2, err2 := SelectItemByWeight(master, table, rng2, 5)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, result1, result2, "同じシードで同じ結果が得られるべき")
	}
}
