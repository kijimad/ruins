package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のoapi.Rawsを作成する。アイテムグループとアイテムコレクションとアイテムテーブルを含む
func newTestRawsForItemTable(groups []oapi.ItemGroup, table oapi.ItemTable) oapi.Raws {
	return oapi.Raws{
		ItemGroups:      &groups,
		ItemCollections: &testCollections,
		ItemTables:      &[]oapi.ItemTable{table},
	}
}

var testGroups = []oapi.ItemGroup{
	{
		Id:   "回復",
		Name: "回復",
		Entries: []oapi.ItemGroupEntry{
			{Id: "回復薬", Weight: 1.0, Pack: "1d1"},
		},
	},
	{
		Id:   "武器",
		Name: "武器",
		Entries: []oapi.ItemGroupEntry{
			{Id: "毒消し", Weight: 0.8, Pack: "1d1"},
			{Id: "手榴弾", Weight: 0.5, Pack: "1d1"},
		},
	},
	{
		Id:   "素材",
		Name: "素材",
		Entries: []oapi.ItemGroupEntry{
			{Id: "アイテム1", Weight: 1.0, Pack: "1d1"},
			{Id: "アイテム2", Weight: 1.0, Pack: "1d1"},
		},
	},
}

var testCollections = []oapi.ItemCollection{
	{
		Id:   "がらくた",
		Name: "がらくた",
		Entries: []oapi.ItemCollectionEntry{
			{Id: "鉄くず", Chance: 100, Pack: "3d1"},
			{Id: "歯車", Chance: 100, Pack: "1d1"},
			{Id: "骨", Chance: 0, Pack: "1d1"},
		},
	},
}

func singleTable(groupID string) oapi.ItemTable {
	return oapi.ItemTable{
		Name:    "テスト",
		Entries: []oapi.ItemTableEntry{{Id: groupID, Weight: 1.0, MinDanger: 1, MaxDanger: 20}},
	}
}

func TestDrawFromItemTable_グループは1つ引く(t *testing.T) {
	t.Parallel()

	table := singleTable("回復")
	raws := newTestRawsForItemTable(testGroups, table)

	got, err := DrawFromItemTable(raws, table, rand.New(rand.NewPCG(12345, 67890)), 5)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "回復薬", Count: 1}}, got)
}

func TestDrawFromItemTable_コレクションは各行を確率で独立に引きpackの個数を出す(t *testing.T) {
	t.Parallel()

	table := singleTable("がらくた")
	raws := newTestRawsForItemTable(testGroups, table)

	got, err := DrawFromItemTable(raws, table, rand.New(rand.NewPCG(1, 2)), 5)
	require.NoError(t, err)
	assert.Equal(t, []DrawnItem{{Name: "鉄くず", Count: 3}, {Name: "歯車", Count: 1}}, got)
}

func TestDrawFromItemTable_危険度が最小未満はエラー(t *testing.T) {
	t.Parallel()

	table := singleTable("回復")
	raws := newTestRawsForItemTable(testGroups, table)
	_, err := DrawFromItemTable(raws, table, rand.New(rand.NewPCG(1, 2)), MinDanger-1)
	require.ErrorIs(t, err, errDangerBelowMinimum)
}

func TestDrawFromItemTable_複数のグループから引く(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name: "通常",
		Entries: []oapi.ItemTableEntry{
			{Id: "回復", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
			{Id: "武器", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
		},
	}
	raws := newTestRawsForItemTable(testGroups, table)

	seen := map[string]bool{}
	rng := rand.New(rand.NewPCG(12345, 67890))
	for range 1000 {
		got, err := DrawFromItemTable(raws, table, rng, 5)
		require.NoError(t, err)
		for _, d := range got {
			seen[d.Name] = true
		}
	}
	assert.Equal(t, map[string]bool{"回復薬": true, "毒消し": true, "手榴弾": true}, seen)
}

func TestDrawFromItemTable_選べるグループが無ければ何も出ない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []oapi.ItemTableEntry
	}{
		{"重みが全て0", []oapi.ItemTableEntry{
			{Id: "回復", Weight: 0, MinDanger: 1, MaxDanger: 10},
			{Id: "武器", Weight: 0, MinDanger: 1, MaxDanger: 10},
		}},
		{"エントリが空", []oapi.ItemTableEntry{}},
		{"危険度の範囲外", []oapi.ItemTableEntry{{Id: "武器", Weight: 1.0, MinDanger: 10, MaxDanger: 20}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			table := oapi.ItemTable{Name: "テスト", Entries: tt.entries}
			raws := newTestRawsForItemTable(testGroups, table)
			got, err := DrawFromItemTable(raws, table, rand.New(rand.NewPCG(1, 2)), 5)
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	}
}

func TestDrawFromItemTable_グループ未存在はエラー(t *testing.T) {
	t.Parallel()

	table := singleTable("存在しないグループ")
	raws := newTestRawsForItemTable(testGroups, table)
	_, err := DrawFromItemTable(raws, table, rand.New(rand.NewPCG(1, 2)), 5)
	require.ErrorIs(t, err, errItemGroupNotExist)
}

func TestDrawFromItemTable_同じシードで同じ結果(t *testing.T) {
	t.Parallel()

	table := oapi.ItemTable{
		Name: "通常",
		Entries: []oapi.ItemTableEntry{
			{Id: "回復", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
			{Id: "武器", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
			{Id: "素材", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
		},
	}
	raws := newTestRawsForItemTable(testGroups, table)

	rng1 := rand.New(rand.NewPCG(99999, 100000))
	rng2 := rand.New(rand.NewPCG(99999, 100000))
	for range 100 {
		got1, err1 := DrawFromItemTable(raws, table, rng1, 5)
		got2, err2 := DrawFromItemTable(raws, table, rng2, 5)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, got1, got2)
	}
}
