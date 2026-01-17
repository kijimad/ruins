package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCalculateMaxCarryingWeight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		attributes *gc.Attributes
		expected   float64
	}{
		{
			name:       "nil attributes",
			attributes: nil,
			expected:   baseCarryingWeight,
		},
		{
			name: "strength 0",
			attributes: &gc.Attributes{
				Strength: gc.Attribute{Base: 0},
			},
			expected: baseCarryingWeight, // 10.0
		},
		{
			name: "strength 5",
			attributes: &gc.Attributes{
				Strength: gc.Attribute{Base: 5},
			},
			expected: baseCarryingWeight + 5*strengthWeightMultiplier, // 10 + 10 = 20
		},
		{
			name: "strength 10",
			attributes: &gc.Attributes{
				Strength: gc.Attribute{Base: 10},
			},
			expected: baseCarryingWeight + 10*strengthWeightMultiplier, // 10 + 20 = 30
		},
		{
			name: "strength with modifier",
			attributes: &gc.Attributes{
				Strength: gc.Attribute{Base: 5, Modifier: 3}, // Total: 8
			},
			expected: baseCarryingWeight + 8*strengthWeightMultiplier, // 10 + 16 = 26
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := calculateMaxCarryingWeight(tt.attributes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateCurrentCarryingWeight(t *testing.T) {
	t.Parallel()
	t.Run("バックパック内の単一アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		// 1kgのアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.0})
		item.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

		weight := calculateCurrentCarryingWeight(world, player)
		assert.Equal(t, 1.0, weight)
	})

	t.Run("バックパック内の複数アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		// 1kgと2kgのアイテムを作成
		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item1.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.0})
		item1.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item2.AddComponent(world.Components.Weight, &gc.Weight{Kg: 2.0})
		item2.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

		weight := calculateCurrentCarryingWeight(world, player)
		assert.Equal(t, 3.0, weight)
	})

	t.Run("スタック可能アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		// 0.5kg × 5個のスタックアイテム
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 5})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 0.5})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{})
		item.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

		weight := calculateCurrentCarryingWeight(world, player)
		assert.Equal(t, 2.5, weight)
	})

	t.Run("装備中のアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		// 3kgの装備アイテム
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateCurrentCarryingWeight(world, player)
		assert.Equal(t, 3.0, weight)
	})

	t.Run("バックパックと装備の合計", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		// バックパック内: 1kg
		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item1.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.0})
		item1.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

		// 装備中: 3kg
		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item2.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item2.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateCurrentCarryingWeight(world, player)
		assert.Equal(t, 4.0, weight)
	})

	t.Run("他のプレイヤーの装備は含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player1 := world.Manager.NewEntity()
		player2 := world.Manager.NewEntity()

		// player2が装備している3kgのアイテム
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{
			Owner:         player2,
			EquipmentSlot: gc.SlotHead,
		})

		// player1の所持重量は0であるべき
		weight := calculateCurrentCarryingWeight(world, player1)
		assert.Equal(t, 0.0, weight)
	})

	t.Run("フィールド上のアイテムは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		// フィールド上の5kgのアイテム
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 5.0})
		item.AddComponent(world.Components.ItemLocationOnField, &gc.LocationOnField{})

		weight := calculateCurrentCarryingWeight(world, player)
		assert.Equal(t, 0.0, weight)
	})
}

func TestUpdateCarryingWeight(t *testing.T) {
	t.Parallel()
	t.Run("Poolsとattributesがある場合に更新される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Pools, &gc.Pools{})
		player.AddComponent(world.Components.Attributes, &gc.Attributes{
			Strength: gc.Attribute{Base: 10},
		})

		// 2kgのアイテムを追加
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 2.0})
		item.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

		UpdateCarryingWeight(world, player)

		pools := world.Components.Pools.Get(player).(*gc.Pools)
		assert.Equal(t, 30.0, pools.Weight.Max)    // 10 + 10*2
		assert.Equal(t, 2.0, pools.Weight.Current) // 2kg
	})

	t.Run("Poolsがない場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Attributes, &gc.Attributes{})

		// パニックしないことを確認
		UpdateCarryingWeight(world, player)
	})

	t.Run("Attributesがない場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Pools, &gc.Pools{})

		// パニックしないことを確認
		UpdateCarryingWeight(world, player)
	})
}
