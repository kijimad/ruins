package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipMenuWearSortIntegration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state := &EquipMenuState{}

	// テスト用防具エンティティを作成（名前順ではない順序で作成）
	wearable1 := world.Manager.NewEntity()
	wearable1.AddComponent(world.Components.Name, &gc.Name{Name: "Shield"})
	wearable1.AddComponent(world.Components.Item, &gc.Item{})
	wearable1.AddComponent(world.Components.ItemLocationInPlayerBackpack, gc.LocationInPlayerBackpack{})
	wearable1.AddComponent(world.Components.Wearable, &gc.Wearable{EquipmentCategory: gc.EquipmentHead})

	wearable2 := world.Manager.NewEntity()
	wearable2.AddComponent(world.Components.Name, &gc.Name{Name: "Armor"})
	wearable2.AddComponent(world.Components.Item, &gc.Item{})
	wearable2.AddComponent(world.Components.ItemLocationInPlayerBackpack, gc.LocationInPlayerBackpack{})
	wearable2.AddComponent(world.Components.Wearable, &gc.Wearable{EquipmentCategory: gc.EquipmentHead})

	wearable3 := world.Manager.NewEntity()
	wearable3.AddComponent(world.Components.Name, &gc.Name{Name: "Helmet"})
	wearable3.AddComponent(world.Components.Item, &gc.Item{})
	wearable3.AddComponent(world.Components.ItemLocationInPlayerBackpack, gc.LocationInPlayerBackpack{})
	wearable3.AddComponent(world.Components.Wearable, &gc.Wearable{EquipmentCategory: gc.EquipmentHead})

	// queryEquipableItemsForSlotのテスト（頭部装備）
	wearables := state.queryEquipableItemsForSlot(world, gc.SlotHead)
	require.Len(t, wearables, 3, "防具が3つ見つかるべき")

	// ソート順を確認（名前順）
	name1 := world.Components.Name.Get(wearables[0]).(*gc.Name)
	name2 := world.Components.Name.Get(wearables[1]).(*gc.Name)
	name3 := world.Components.Name.Get(wearables[2]).(*gc.Name)

	assert.Equal(t, "Armor", name1.Name, "1番目の防具名が正しくない")
	assert.Equal(t, "Helmet", name2.Name, "2番目の防具名が正しくない")
	assert.Equal(t, "Shield", name3.Name, "3番目の防具名が正しくない")

	// クリーンアップ
	world.Manager.DeleteEntity(wearable1)
	world.Manager.DeleteEntity(wearable2)
	world.Manager.DeleteEntity(wearable3)
}

func TestEquipMenuWeaponSortIntegration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state := &EquipMenuState{}

	// テスト用武器エンティティを作成（名前順ではない順序で作成）
	weapon1 := world.Manager.NewEntity()
	weapon1.AddComponent(world.Components.Name, &gc.Name{Name: "Thunder Weapon"})
	weapon1.AddComponent(world.Components.Item, &gc.Item{})
	weapon1.AddComponent(world.Components.ItemLocationInPlayerBackpack, gc.LocationInPlayerBackpack{})
	weapon1.AddComponent(world.Components.Weapon, &gc.Weapon{})
	weapon1.AddComponent(world.Components.Attack, &gc.Attack{AttackCategory: gc.AttackSword})

	weapon2 := world.Manager.NewEntity()
	weapon2.AddComponent(world.Components.Name, &gc.Name{Name: "Fire Weapon"})
	weapon2.AddComponent(world.Components.Item, &gc.Item{})
	weapon2.AddComponent(world.Components.ItemLocationInPlayerBackpack, gc.LocationInPlayerBackpack{})
	weapon2.AddComponent(world.Components.Weapon, &gc.Weapon{})
	weapon2.AddComponent(world.Components.Attack, &gc.Attack{AttackCategory: gc.AttackSpear})

	weapon3 := world.Manager.NewEntity()
	weapon3.AddComponent(world.Components.Name, &gc.Name{Name: "Ice Weapon"})
	weapon3.AddComponent(world.Components.Item, &gc.Item{})
	weapon3.AddComponent(world.Components.ItemLocationInPlayerBackpack, gc.LocationInPlayerBackpack{})
	weapon3.AddComponent(world.Components.Weapon, &gc.Weapon{})
	weapon3.AddComponent(world.Components.Attack, &gc.Attack{AttackCategory: gc.AttackFist})

	// queryEquipableItemsForSlotのテスト（近接武器）
	weapons := state.queryEquipableItemsForSlot(world, gc.SlotMeleeWeapon)
	require.Len(t, weapons, 3, "武器が3つ見つかるべき")

	// ソート順を確認（名前順）
	name1 := world.Components.Name.Get(weapons[0]).(*gc.Name)
	name2 := world.Components.Name.Get(weapons[1]).(*gc.Name)
	name3 := world.Components.Name.Get(weapons[2]).(*gc.Name)

	assert.Equal(t, "Fire Weapon", name1.Name, "1番目の武器名が正しくない")
	assert.Equal(t, "Ice Weapon", name2.Name, "2番目の武器名が正しくない")
	assert.Equal(t, "Thunder Weapon", name3.Name, "3番目の武器名が正しくない")

	// クリーンアップ
	world.Manager.DeleteEntity(weapon1)
	world.Manager.DeleteEntity(weapon2)
	world.Manager.DeleteEntity(weapon3)
}
