package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestWeightDirtySystem(t *testing.T) {
	t.Parallel()

	t.Run("マーカー付きエンティティの重量を再計算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		// WeightCapacityを持つStorageエンティティを作成
		storage := world.Manager.NewEntity()
		storage.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 50.0})

		// アイテムを収納に入れる
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 3.0})
		item.AddComponent(world.Components.LocationInStorage, &gc.LocationInStorage{Owner: storage})

		// WeightDirtyマーカーを付与
		storage.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

		err := sys.Update(world)
		assert.NoError(t, err)

		// Currentが再計算されている
		wc := world.Components.WeightCapacity.Get(storage).(*gc.WeightCapacity)
		assert.Equal(t, 3.0, wc.Current)
	})

	t.Run("マーカーが処理後にクリアされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 10.0})
		entity.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

		err := sys.Update(world)
		assert.NoError(t, err)

		assert.False(t, entity.HasComponent(world.Components.WeightDirty), "マーカーはクリアされるべき")
	})

	t.Run("複数エンティティのマーカーを一括処理する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		// Player
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{})
		player.AddComponent(world.Components.Abilities, &gc.Abilities{Strength: gc.Ability{Base: 5}})
		player.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

		backpackItem := world.Manager.NewEntity()
		backpackItem.AddComponent(world.Components.Weight, &gc.Weight{Kg: 2.0})
		backpackItem.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})

		// Storage
		storage := world.Manager.NewEntity()
		storage.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 30.0})
		storage.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

		storageItem := world.Manager.NewEntity()
		storageItem.AddComponent(world.Components.Weight, &gc.Weight{Kg: 5.0})
		storageItem.AddComponent(world.Components.LocationInStorage, &gc.LocationInStorage{Owner: storage})

		err := sys.Update(world)
		assert.NoError(t, err)

		playerWc := world.Components.WeightCapacity.Get(player).(*gc.WeightCapacity)
		assert.Equal(t, 20.0, playerWc.Max)    // 10 + 5*2
		assert.Equal(t, 2.0, playerWc.Current) // バックパック内2kg

		storageWc := world.Components.WeightCapacity.Get(storage).(*gc.WeightCapacity)
		assert.Equal(t, 30.0, storageWc.Max)    // 変更されない
		assert.Equal(t, 5.0, storageWc.Current) // 収納内5kg

		assert.False(t, player.HasComponent(world.Components.WeightDirty))
		assert.False(t, storage.HasComponent(world.Components.WeightDirty))
	})

	t.Run("マーカーがなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Max: 10.0, Current: 99.0})

		err := sys.Update(world)
		assert.NoError(t, err)

		// Currentは変わらない
		wc := world.Components.WeightCapacity.Get(entity).(*gc.WeightCapacity)
		assert.Equal(t, 99.0, wc.Current, "マーカーがないので再計算されない")
	})
}
