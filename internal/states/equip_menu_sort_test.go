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
	wearable1 := world.World.NewEntity()
	world.Components.Name.Add(wearable1, &gc.Name{Name: "Shield"})
	world.Components.LocationInBackpack.Add(wearable1, &gc.LocationInBackpack{})
	world.Components.Wearable.Add(wearable1, &gc.Wearable{EquipmentCategory: gc.EquipmentHead})

	wearable2 := world.World.NewEntity()
	world.Components.Name.Add(wearable2, &gc.Name{Name: "Armor"})
	world.Components.LocationInBackpack.Add(wearable2, &gc.LocationInBackpack{})
	world.Components.Wearable.Add(wearable2, &gc.Wearable{EquipmentCategory: gc.EquipmentHead})

	wearable3 := world.World.NewEntity()
	world.Components.Name.Add(wearable3, &gc.Name{Name: "Helmet"})
	world.Components.LocationInBackpack.Add(wearable3, &gc.LocationInBackpack{})
	world.Components.Wearable.Add(wearable3, &gc.Wearable{EquipmentCategory: gc.EquipmentHead})

	// queryEquipableItemsForSlotのテスト（頭部装備）
	wearables := state.queryEquipableItemsForSlot(world, gc.SlotHead)
	require.Len(t, wearables, 3, "防具が3つ見つかるべき")

	// ソート順を確認（名前順）
	name1 := world.Components.Name.Get(wearables[0])
	name2 := world.Components.Name.Get(wearables[1])
	name3 := world.Components.Name.Get(wearables[2])

	assert.Equal(t, "Armor", name1.Name, "1番目の防具名が正しくない")
	assert.Equal(t, "Helmet", name2.Name, "2番目の防具名が正しくない")
	assert.Equal(t, "Shield", name3.Name, "3番目の防具名が正しくない")

	// クリーンアップ
	world.World.RemoveEntity(wearable1)
	world.World.RemoveEntity(wearable2)
	world.World.RemoveEntity(wearable3)
}

func TestEquipMenuWeaponSortIntegration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	state := &EquipMenuState{}

	// テスト用武器エンティティを作成（名前順ではない順序で作成）
	weapon1 := world.World.NewEntity()
	world.Components.Name.Add(weapon1, &gc.Name{Name: "Thunder Weapon"})
	world.Components.LocationInBackpack.Add(weapon1, &gc.LocationInBackpack{})
	world.Components.Melee.Add(weapon1, &gc.Melee{AttackCategory: gc.AttackSword})

	weapon2 := world.World.NewEntity()
	world.Components.Name.Add(weapon2, &gc.Name{Name: "Fire Weapon"})
	world.Components.LocationInBackpack.Add(weapon2, &gc.LocationInBackpack{})
	world.Components.Melee.Add(weapon2, &gc.Melee{AttackCategory: gc.AttackSpear})

	weapon3 := world.World.NewEntity()
	world.Components.Name.Add(weapon3, &gc.Name{Name: "Ice Weapon"})
	world.Components.LocationInBackpack.Add(weapon3, &gc.LocationInBackpack{})
	world.Components.Melee.Add(weapon3, &gc.Melee{AttackCategory: gc.AttackFist})

	// queryEquipableItemsForSlotのテスト（武器スロット1）
	weapons := state.queryEquipableItemsForSlot(world, gc.SlotWeapon1)
	require.Len(t, weapons, 3, "武器が3つ見つかるべき")

	// ソート順を確認（名前順）
	name1 := world.Components.Name.Get(weapons[0])
	name2 := world.Components.Name.Get(weapons[1])
	name3 := world.Components.Name.Get(weapons[2])

	assert.Equal(t, "Fire Weapon", name1.Name, "1番目の武器名が正しくない")
	assert.Equal(t, "Ice Weapon", name2.Name, "2番目の武器名が正しくない")
	assert.Equal(t, "Thunder Weapon", name3.Name, "3番目の武器名が正しくない")

	// クリーンアップ
	world.World.RemoveEntity(weapon1)
	world.World.RemoveEntity(weapon2)
	world.World.RemoveEntity(weapon3)
}
