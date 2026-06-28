package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetEntityWeight(t *testing.T) {
	t.Parallel()

	t.Run("Weightなしのエンティティは0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.Manager.NewEntity()
		assert.Equal(t, 0.0, GetEntityWeight(world, e))
	})

	t.Run("単体アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.Manager.NewEntity()
		e.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.5})
		assert.Equal(t, 1.5, GetEntityWeight(world, e))
	})

	t.Run("スタックアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.Manager.NewEntity()
		e.AddComponent(world.Components.Weight, &gc.Weight{Kg: 0.5})
		e.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 3})
		assert.Equal(t, 1.5, GetEntityWeight(world, e))
	})
}

func TestCalculateOwnedWeight(t *testing.T) {
	t.Parallel()
	t.Run("バックパック内の単一アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.0})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, 1.0, weight)
	})

	t.Run("バックパック内の複数アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.0})
		item1.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Weight, &gc.Weight{Kg: 2.0})
		item2.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, 3.0, weight)
	})

	t.Run("スタック可能アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 0.5})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 5})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, 2.5, weight)
	})

	t.Run("装備中のアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, 3.0, weight)
	})

	t.Run("バックパックと装備の合計", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Weight, &gc.Weight{Kg: 1.0})
		item1.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item2.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, 4.0, weight)
	})

	t.Run("他のプレイヤーの装備は含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player1 := world.Manager.NewEntity()
		player2 := world.Manager.NewEntity()

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
			Owner:         player2,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateOwnedWeight(world, player1)
		assert.Equal(t, 0.0, weight)
	})

	t.Run("フィールド上のアイテムは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 5.0})
		item.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, 0.0, weight)
	})

	t.Run("収納内のアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := world.Manager.NewEntity()

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 2.0})
		item.AddComponent(world.Components.LocationInStorage, &gc.LocationInStorage{Owner: storage})

		weight := calculateOwnedWeight(world, storage)
		assert.Equal(t, 2.0, weight)
	})
}

func TestUpdateWeightCapacity(t *testing.T) {
	t.Parallel()
	t.Run("Playerの場合はAbilitiesからMaxを計算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{})
		player.AddComponent(world.Components.Abilities, &gc.Abilities{
			Strength: gc.Ability{Base: 10},
		})

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 2.0})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		UpdateWeightCapacity(world, player)

		wc := world.Components.WeightCapacity.Get(player).(*gc.WeightCapacity)
		assert.Equal(t, 30.0, wc.Max)    // 10 + 10*2
		assert.Equal(t, 2.0, wc.Current) // 2kg
	})

	t.Run("Storageの場合はMaxを変更しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := world.Manager.NewEntity()
		storage.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 20.0})

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item.AddComponent(world.Components.LocationInStorage, &gc.LocationInStorage{Owner: storage})

		UpdateWeightCapacity(world, storage)

		wc := world.Components.WeightCapacity.Get(storage).(*gc.WeightCapacity)
		assert.Equal(t, 20.0, wc.Max)    // 変更されない
		assert.Equal(t, 3.0, wc.Current) // 3kg
	})

	t.Run("CharModifiersによるMax倍率が適用される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{})
		player.AddComponent(world.Components.Abilities, &gc.Abilities{
			Strength: gc.Ability{Base: 10},
		})
		// MaxWeight=150 → 基本Max(30.0) * 150/100 = 45.0
		player.AddComponent(world.Components.CharModifiers, &gc.CharModifiers{
			MaxWeight: 150,
		})

		UpdateWeightCapacity(world, player)

		wc := world.Components.WeightCapacity.Get(player).(*gc.WeightCapacity)
		assert.Equal(t, 45.0, wc.Max)
	})

	t.Run("WeightCapacityがない場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Abilities, &gc.Abilities{})

		UpdateWeightCapacity(world, entity)
	})
}
