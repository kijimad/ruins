package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestEquipDisarm(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// アイテムエンティティを作成
	item := world.Manager.NewEntity()
	item.AddComponent(world.Components.Item, &gc.Item{})
	item.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.ItemLocationInPlayerBackpack)

	// オーナーエンティティを作成
	owner := world.Manager.NewEntity()

	// 装備する
	Equip(world, item, owner, gc.EquipmentSlotNumber(0))

	// 装備されたことを確認
	assert.True(t, item.HasComponent(world.Components.ItemLocationEquipped), "アイテムが装備されていない")
	assert.False(t, item.HasComponent(world.Components.ItemLocationInPlayerBackpack), "アイテムがまだバックパックにある")
	assert.True(t, item.HasComponent(world.Components.EquipmentChanged), "装備変更フラグが設定されていない")

	equipped := world.Components.ItemLocationEquipped.Get(item).(*gc.LocationEquipped)
	assert.Equal(t, owner, equipped.Owner, "オーナーが正しく設定されていない")
	assert.Equal(t, gc.EquipmentSlotNumber(0), equipped.EquipmentSlot, "スロット番号が正しく設定されていない")

	// 装備を外す
	Disarm(world, item)

	// 装備が外されたことを確認
	assert.False(t, item.HasComponent(world.Components.ItemLocationEquipped), "アイテムがまだ装備されている")
	assert.True(t, item.HasComponent(world.Components.ItemLocationInPlayerBackpack), "アイテムがバックパックに戻っていない")
	assert.True(t, item.HasComponent(world.Components.EquipmentChanged), "装備変更フラグが設定されていない")

	// クリーンアップ
	world.Manager.DeleteEntity(item)
	world.Manager.DeleteEntity(owner)
}
