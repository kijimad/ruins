package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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
		storage := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{Max: consts.MustParseWeight("50 kg")})

		// アイテムを収納に入れる
		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("3 kg")})
		world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: storage})

		// WeightDirtyマーカーを付与
		world.Components.WeightDirty.Add(storage, &gc.WeightDirty{})

		err := sys.Update(world)
		require.NoError(t, err)

		// Currentが再計算されている
		wc := world.Components.WeightCapacity.Get(storage)
		assert.Equal(t, consts.MustParseWeight("3 kg"), wc.Current)
	})

	t.Run("マーカーが処理後にクリアされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		entity := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{Max: consts.MustParseWeight("10 kg")})
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
		player := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})
		world.Components.Abilities.Add(player, &gc.Abilities{Strength: gc.Ability{Base: 5}})
		world.Components.WeightDirty.Add(player, &gc.WeightDirty{})

		backpackItem := world.ECS.NewEntity()
		world.Components.Weight.Add(backpackItem, &gc.Weight{Milligram: consts.MustParseWeight("2 kg")})
		world.Components.LocationInBackpack.Add(backpackItem, &gc.LocationInBackpack{Owner: player})

		// Storage
		storage := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{Max: consts.MustParseWeight("30 kg")})
		world.Components.WeightDirty.Add(storage, &gc.WeightDirty{})

		storageItem := world.ECS.NewEntity()
		world.Components.Weight.Add(storageItem, &gc.Weight{Milligram: consts.MustParseWeight("5 kg")})
		world.Components.LocationInStorage.Add(storageItem, &gc.LocationInStorage{Owner: storage})

		err := sys.Update(world)
		require.NoError(t, err)

		playerWc := world.Components.WeightCapacity.Get(player)
		assert.Equal(t, consts.MustParseWeight("20 kg"), playerWc.Max)    // 10 + 5*2
		assert.Equal(t, consts.MustParseWeight("2 kg"), playerWc.Current) // バックパック内2kg

		storageWc := world.Components.WeightCapacity.Get(storage)
		assert.Equal(t, consts.MustParseWeight("30 kg"), storageWc.Max)    // 変更されない
		assert.Equal(t, consts.MustParseWeight("5 kg"), storageWc.Current) // 収納内5kg

		assert.False(t, world.Components.WeightDirty.Has(player))
		assert.False(t, world.Components.WeightDirty.Has(storage))
	})

	t.Run("マーカーがなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &WeightDirtySystem{}

		entity := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{
			Max:     consts.MustParseWeight("10 kg"),
			Current: consts.MustParseWeight("99 kg"),
		})

		err := sys.Update(world)
		require.NoError(t, err)

		// Currentは変わらない
		wc := world.Components.WeightCapacity.Get(entity)
		assert.Equal(t, consts.MustParseWeight("99 kg"), wc.Current, "マーカーがないので再計算されない")
	})
}
