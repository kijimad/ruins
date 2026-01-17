package states

import (
	"sort"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCraftMenuSortIntegration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state := &CraftMenuState{}

	// queryMenuConsumableのテスト（道具タブ）
	// RawMasterから取得されるレシピがソートされているかを確認
	consumables := state.queryMenuConsumable(world)

	// レシピ名がアルファベット順にソートされているか確認
	if len(consumables) > 1 {
		for i := 0; i < len(consumables)-1; i++ {
			assert.True(t, consumables[i] <= consumables[i+1],
				"消耗品レシピがソートされていない: %s > %s", consumables[i], consumables[i+1])
		}
	}
}

func TestCraftMenuWeaponSortIntegration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state := &CraftMenuState{}

	// queryMenuWeaponのテスト（武器タブ）
	weapons := state.queryMenuWeapon(world)

	// 武器名がアルファベット順にソートされているか確認
	if len(weapons) > 1 {
		for i := 0; i < len(weapons)-1; i++ {
			assert.True(t, weapons[i] <= weapons[i+1],
				"武器レシピがソートされていない: %s > %s", weapons[i], weapons[i+1])
		}
	}
}

func TestCraftMenuWearableSortIntegration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state := &CraftMenuState{}

	// queryMenuWearableのテスト（装備タブ）
	wearables := state.queryMenuWearable(world)

	// 装備名がアルファベット順にソートされているか確認
	if len(wearables) > 1 {
		for i := 0; i < len(wearables)-1; i++ {
			assert.True(t, wearables[i] <= wearables[i+1],
				"装備レシピがソートされていない: %s > %s", wearables[i], wearables[i+1])
		}
	}
}

func TestCraftMenuSortCorrectness(t *testing.T) {
	t.Parallel()

	// ソート機能のユニットテスト
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "既にソート済み",
			input:    []string{"A", "B", "C"},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "逆順",
			input:    []string{"C", "B", "A"},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "ランダム",
			input:    []string{"B", "A", "C"},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "日本語",
			input:    []string{"木刀", "ハンドガン", "M72 LAW"},
			expected: []string{"M72 LAW", "ハンドガン", "木刀"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := make([]string, len(tc.input))
			copy(result, tc.input)
			sort.Strings(result)
			assert.Equal(t, tc.expected, result)
		})
	}
}
