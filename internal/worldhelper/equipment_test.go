package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWeapons_Empty(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// 初期状態は全てnil
	weapons := GetWeapons(world, player)
	assert.Len(t, weapons, 5)
	for _, w := range weapons {
		assert.Nil(t, w)
	}
}

func TestGetArmorEquipments(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// 初期状態は全てnil
	armors := GetArmorEquipments(world, player)
	assert.Len(t, armors, 7)

	// 防具を装備する
	armor := world.Manager.NewEntity()
	armor.AddComponent(world.Components.Name, &gc.Name{Name: "テスト鎧"})
	armor.AddComponent(world.Components.Wearable, &gc.Wearable{
		EquipmentCategory: gc.EquipmentTorso,
		Defense:           5,
	})
	MoveToEquip(world, armor, player, gc.SlotTorso)

	armors = GetArmorEquipments(world, player)
	assert.NotNil(t, armors[1], "SlotTorsoに装備が入っている")
	assert.Nil(t, armors[0], "SlotHeadは空")
}

func TestEquipDisarm(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// アイテムエンティティを作成
	item := world.Manager.NewEntity()
	item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})

	// オーナーエンティティを作成
	owner := world.Manager.NewEntity()

	// 装備する
	MoveToEquip(world, item, owner, gc.EquipmentSlotNumber(0))

	// 装備されたことを確認
	assert.True(t, item.HasComponent(world.Components.LocationEquipped), "アイテムが装備されていない")
	assert.False(t, item.HasComponent(world.Components.LocationInBackpack), "アイテムがまだバックパックにある")
	assert.True(t, owner.HasComponent(world.Components.StatsChanged), "オーナーにステータス再計算フラグが設定されていない")

	equipped := world.Components.LocationEquipped.Get(item).(*gc.LocationEquipped)
	assert.Equal(t, owner, equipped.Owner, "オーナーが正しく設定されていない")
	assert.Equal(t, gc.EquipmentSlotNumber(0), equipped.EquipmentSlot, "スロット番号が正しく設定されていない")

	// 装備を外す
	require.NoError(t, MoveToBackpack(world, item, owner))

	// 装備が外されたことを確認
	assert.False(t, item.HasComponent(world.Components.LocationEquipped), "アイテムがまだ装備されている")
	assert.True(t, item.HasComponent(world.Components.LocationInBackpack), "アイテムがバックパックに戻っていない")
	assert.True(t, owner.HasComponent(world.Components.StatsChanged), "オーナーにステータス再計算フラグが設定されていない")

	// クリーンアップ
	world.Manager.DeleteEntity(item)
	world.Manager.DeleteEntity(owner)
}
