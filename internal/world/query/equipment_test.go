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

func TestIsWeapon(t *testing.T) {
	t.Parallel()

	t.Run("近接コンポーネントを持てば武器", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Melee.Add(e, &gc.Melee{Accuracy: 80, Damage: 5})
		assert.True(t, query.IsWeapon(world, e))
	})

	t.Run("射撃コンポーネントを持てば武器", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Fire.Add(e, &gc.Fire{Accuracy: 80, Damage: 5})
		assert.True(t, query.IsWeapon(world, e))
	})

	t.Run("どちらも持たなければ武器でない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		assert.False(t, query.IsWeapon(world, e))
	})
}

func TestGetWeapons_Empty(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// SpawnPlayer は初期装備の松明を武器スロットに持つ。空の状態を検証するため一旦外す
	require.NoError(t, lifecycle.UnequipAll(world, player))

	// 装備を外せば全てnil
	weapons := query.GetWeapons(world, player)
	assert.Len(t, weapons, 5)
	for _, w := range weapons {
		assert.Nil(t, w)
	}
}

func TestGetWeapons_装備した武器をスロット順で返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// SpawnPlayer は初期装備の松明をスロット1に武器として装備する
	weapons := query.GetWeapons(world, player)
	require.Len(t, weapons, 5)
	require.NotNil(t, weapons[0], "スロット1に初期装備の松明が入っている")
	assert.True(t, query.IsWeapon(world, *weapons[0]), "松明は近接コンポーネントを持つので武器と判定される")
	for i := 1; i < 5; i++ {
		assert.Nil(t, weapons[i], "スロット%dは空", i+1)
	}
}

func TestGetArmorEquipments(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
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

// TestGetArmorEquipments_不正なスロットはpanic は防具スロットの範囲外(武器スロット)を
// Wearable に割り当てた不整合データを switch の default 分岐に落とし、panic することを固定する。
func TestGetArmorEquipments_不正なスロットはpanic(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	item := world.ECS.NewEntity()
	world.Components.Wearable.Add(item, &gc.Wearable{EquipmentCategory: gc.EquipmentTorso})
	lifecycle.MoveToEquip(world, item, player, gc.SlotWeapon1)

	assert.Panics(t, func() { query.GetArmorEquipments(world, player) })
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
