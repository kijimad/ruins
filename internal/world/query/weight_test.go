package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetEntityWeight(t *testing.T) {
	t.Parallel()

	t.Run("Weightなしのエンティティは0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		assert.Equal(t, consts.Milligram(0), GetEntityWeight(world, e))
	})

	t.Run("単体アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Weight.Add(e, &gc.Weight{Milligram: consts.MustParseWeight("1.5 kg")})
		assert.Equal(t, consts.MustParseWeight("1.5 kg"), GetEntityWeight(world, e))
	})

	t.Run("スタックアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Weight.Add(e, &gc.Weight{Milligram: consts.MustParseWeight("0.5 kg")})
		world.Components.Stackable.Add(e, &gc.Stackable{Count: 3})
		assert.Equal(t, consts.MustParseWeight("1.5 kg"), GetEntityWeight(world, e))
	})
}

func TestCalculateOwnedWeight(t *testing.T) {
	t.Parallel()
	t.Run("バックパック内の単一アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("1 kg")})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: player})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, consts.MustParseWeight("1 kg"), weight)
	})

	t.Run("バックパック内の複数アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()

		item1 := world.ECS.NewEntity()
		world.Components.Weight.Add(item1, &gc.Weight{Milligram: consts.MustParseWeight("1 kg")})
		world.Components.LocationInBackpack.Add(item1, &gc.LocationInBackpack{Owner: player})

		item2 := world.ECS.NewEntity()
		world.Components.Weight.Add(item2, &gc.Weight{Milligram: consts.MustParseWeight("2 kg")})
		world.Components.LocationInBackpack.Add(item2, &gc.LocationInBackpack{Owner: player})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, consts.MustParseWeight("3 kg"), weight)
	})

	t.Run("スタック可能アイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("0.5 kg")})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 5})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: player})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, consts.MustParseWeight("2.5 kg"), weight)
	})

	t.Run("装備中のアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("3 kg")})
		world.Components.LocationEquipped.Add(item, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, consts.MustParseWeight("3 kg"), weight)
	})

	t.Run("バックパックと装備の合計", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()

		item1 := world.ECS.NewEntity()
		world.Components.Weight.Add(item1, &gc.Weight{Milligram: consts.MustParseWeight("1 kg")})
		world.Components.LocationInBackpack.Add(item1, &gc.LocationInBackpack{Owner: player})

		item2 := world.ECS.NewEntity()
		world.Components.Weight.Add(item2, &gc.Weight{Milligram: consts.MustParseWeight("3 kg")})
		world.Components.LocationEquipped.Add(item2, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, consts.MustParseWeight("4 kg"), weight)
	})

	t.Run("他のプレイヤーの装備は含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player1 := world.ECS.NewEntity()
		player2 := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("3 kg")})
		world.Components.LocationEquipped.Add(item, &gc.LocationEquipped{
			Owner:         player2,
			EquipmentSlot: gc.SlotHead,
		})

		weight := calculateOwnedWeight(world, player1)
		assert.Equal(t, consts.Milligram(0), weight)
	})

	t.Run("フィールド上のアイテムは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("5 kg")})
		world.Components.LocationOnField.Add(item, &gc.LocationOnField{})

		weight := calculateOwnedWeight(world, player)
		assert.Equal(t, consts.Milligram(0), weight)
	})

	t.Run("収納内のアイテム", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("2 kg")})
		world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: storage})

		weight := calculateOwnedWeight(world, storage)
		assert.Equal(t, consts.MustParseWeight("2 kg"), weight)
	})
}

func TestUpdateWeightCapacity(t *testing.T) {
	t.Parallel()
	t.Run("Playerの場合はAbilitiesからMaxを計算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Strength: gc.Ability{Base: 10},
		})

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("2 kg")})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: player})

		UpdateWeightCapacity(world, player)

		wc := world.Components.WeightCapacity.Get(player)
		assert.Equal(t, consts.MustParseWeight("30 kg"), wc.Max)    // 10 + 10*2
		assert.Equal(t, consts.MustParseWeight("2 kg"), wc.Current) // 2kg
	})

	t.Run("Storageの場合はMaxを変更しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{Max: consts.MustParseWeight("20 kg")})

		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("3 kg")})
		world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: storage})

		UpdateWeightCapacity(world, storage)

		wc := world.Components.WeightCapacity.Get(storage)
		assert.Equal(t, consts.MustParseWeight("20 kg"), wc.Max)    // 変更されない
		assert.Equal(t, consts.MustParseWeight("3 kg"), wc.Current) // 3kg
	})

	t.Run("CharModifiersによるMax倍率が適用される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Strength: gc.Ability{Base: 10},
		})
		// MaxWeight=150 → 基本Max(30.0) * 150/100 = 45.0
		world.Components.CharModifiers.Add(player, &gc.CharModifiers{
			MaxWeight: 150,
		})

		UpdateWeightCapacity(world, player)

		wc := world.Components.WeightCapacity.Get(player)
		assert.Equal(t, consts.MustParseWeight("45 kg"), wc.Max)
	})

	t.Run("WeightCapacityがない場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Abilities.Add(entity, &gc.Abilities{})

		UpdateWeightCapacity(world, entity)
	})
}
