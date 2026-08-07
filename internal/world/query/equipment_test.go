package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWeapons_Empty(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	// 初期状態は全てnil
	weapons := query.GetWeapons(world, player)
	assert.Len(t, weapons, 5)
	for _, w := range weapons {
		assert.Nil(t, w)
	}
}

func TestGetArmorEquipments(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	// 初期状態は全てnil
	armors := query.GetArmorEquipments(world, player)
	assert.Len(t, armors, 7)

	// 防具を装備する
	armor := world.ECS.NewEntity()
	world.Components.Name.Add(armor, &gc.Name{Name: "テスト鎧"})
	world.Components.Wearable.Add(armor, &gc.Wearable{
		EquipmentCategory: gc.EquipmentTorso,
		Defense:           5,
	})
	lifecycle.MoveToEquip(world, armor, player, gc.SlotTorso)

	armors = query.GetArmorEquipments(world, player)
	assert.NotNil(t, armors[1], "SlotTorsoに装備が入っている")
	assert.Nil(t, armors[0], "SlotHeadは空")
}

func TestEquipDisarm(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// アイテムエンティティを作成
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{})

	// オーナーエンティティを作成
	owner := world.ECS.NewEntity()

	// 装備する
	lifecycle.MoveToEquip(world, item, owner, gc.EquipmentSlotNumber(0))

	// 装備されたことを確認
	assert.True(t, world.Components.LocationEquipped.Has(item), "アイテムが装備されていない")
	assert.False(t, world.Components.LocationInBackpack.Has(item), "アイテムがまだバックパックにある")
	assert.True(t, world.Components.StatsChanged.Has(owner), "オーナーにステータス再計算フラグが設定されていない")

	equipped := world.Components.LocationEquipped.Get(item)
	assert.Equal(t, owner, equipped.Owner, "オーナーが正しく設定されていない")
	assert.Equal(t, gc.EquipmentSlotNumber(0), equipped.EquipmentSlot, "スロット番号が正しく設定されていない")

	// 装備を外す
	require.NoError(t, lifecycle.MoveToBackpack(world, item, owner))

	// 装備が外されたことを確認
	assert.False(t, world.Components.LocationEquipped.Has(item), "アイテムがまだ装備されている")
	assert.True(t, world.Components.LocationInBackpack.Has(item), "アイテムがバックパックに戻っていない")
	assert.True(t, world.Components.StatsChanged.Has(owner), "オーナーにステータス再計算フラグが設定されていない")

	// クリーンアップ
	world.ECS.RemoveEntity(item)
	world.ECS.RemoveEntity(owner)
}
