package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeightDirtySystem(t *testing.T) {
	t.Parallel()

	t.Run("マーカー付きエンティティの重量を再計算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		// WeightCapacityを持つStorageエンティティを作成
		storage := world.World.NewEntity()
		world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{Max: 50.0})

		// アイテムを収納に入れる
		item := world.World.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Kg: 3.0})
		world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: storage})

		// WeightDirtyマーカーを付与
		world.Components.WeightDirty.Add(storage, &gc.WeightDirty{})

		err := sys.Update(world)
		require.NoError(t, err)

		// Currentが再計算されている
		wc := world.Components.WeightCapacity.Get(storage)
		assert.Equal(t, 3.0, wc.Current)
	})

	t.Run("マーカーが処理後にクリアされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		entity := world.World.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 10.0})
		world.Components.WeightDirty.Add(entity, &gc.WeightDirty{})

		err := sys.Update(world)
		require.NoError(t, err)

		assert.False(t, world.Components.WeightDirty.Has(entity), "マーカーはクリアされるべき")
	})

	t.Run("複数エンティティのマーカーを一括処理する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		// Player
		player := world.World.NewEntity()
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})
		world.Components.Abilities.Add(player, &gc.Abilities{Strength: gc.Ability{Base: 5}})
		world.Components.WeightDirty.Add(player, &gc.WeightDirty{})

		backpackItem := world.World.NewEntity()
		world.Components.Weight.Add(backpackItem, &gc.Weight{Kg: 2.0})
		world.Components.LocationInBackpack.Add(backpackItem, &gc.LocationInBackpack{Owner: player})

		// Storage
		storage := world.World.NewEntity()
		world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{Max: 30.0})
		world.Components.WeightDirty.Add(storage, &gc.WeightDirty{})

		storageItem := world.World.NewEntity()
		world.Components.Weight.Add(storageItem, &gc.Weight{Kg: 5.0})
		world.Components.LocationInStorage.Add(storageItem, &gc.LocationInStorage{Owner: storage})

		err := sys.Update(world)
		require.NoError(t, err)

		playerWc := world.Components.WeightCapacity.Get(player)
		assert.Equal(t, 20.0, playerWc.Max)    // 10 + 5*2
		assert.Equal(t, 2.0, playerWc.Current) // バックパック内2kg

		storageWc := world.Components.WeightCapacity.Get(storage)
		assert.Equal(t, 30.0, storageWc.Max)    // 変更されない
		assert.Equal(t, 5.0, storageWc.Current) // 収納内5kg

		assert.False(t, world.Components.WeightDirty.Has(player))
		assert.False(t, world.Components.WeightDirty.Has(storage))
	})

	t.Run("マーカーがなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		entity := world.World.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: 10.0, Current: 99.0})

		err := sys.Update(world)
		require.NoError(t, err)

		// Currentは変わらない
		wc := world.Components.WeightCapacity.Get(entity)
		assert.Equal(t, 99.0, wc.Current, "マーカーがないので再計算されない")
	})
}
