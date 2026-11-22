package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindStackInInventory_名前が一致するバックパック内アイテムを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})
	world.Components.RawID.Add(item, &gc.RawID{ID: "回復薬"})

	got, found := query.FindStackInInventory(world, "回復薬")
	assert.True(t, found)
	assert.Equal(t, item, got)
}

func TestFindStackInInventory_名前が一致しなければ見つからない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

	_, found := query.FindStackInInventory(world, "毒薬")
	assert.False(t, found)
}

func TestFindStackInInventory_スタックでなければ対象外(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

	_, found := query.FindStackInInventory(world, "回復薬")
	assert.False(t, found)
}

func TestFindStackInInventory_バックパック内でなければ対象外(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

	_, found := query.FindStackInInventory(world, "回復薬")
	assert.False(t, found)
}

func TestFindAmmoInInventory_口径タグが一致するバックパック内弾薬を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	ammo := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(ammo, &gc.LocationInBackpack{Owner: owner})
	world.Components.Ammo.Add(ammo, &gc.Ammo{AmmoTag: rawschema.AmmoTagN9mm})

	got, found := query.FindAmmoInInventory(world, rawschema.AmmoTagN9mm)
	assert.True(t, found)
	assert.Equal(t, ammo, got)
}

func TestFindAmmoInInventory_口径タグが一致しなければ見つからない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	ammo := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(ammo, &gc.LocationInBackpack{Owner: owner})
	world.Components.Ammo.Add(ammo, &gc.Ammo{AmmoTag: rawschema.AmmoTagN9mm})

	_, found := query.FindAmmoInInventory(world, rawschema.AmmoTagRifle)
	assert.False(t, found)
}

func TestFindAmmoInInventory_該当が無ければゼロ値エンティティを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	got, found := query.FindAmmoInInventory(world, rawschema.AmmoTagShell)
	assert.False(t, found)
	assert.Zero(t, got)
}

func TestFindStackInInventory_使いかけの物は選ばず新品を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	stick := func(current int) {
		e := world.ECS.NewEntity()
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: owner})
		world.Components.RawID.Add(e, &gc.RawID{ID: "wooden_stick"})
		world.Components.Charge.Add(e, &gc.Charge{Max: 80, Current: current})
	}
	stick(40)
	_, found := query.FindStackInInventory(world, "wooden_stick")
	assert.False(t, found, "使いかけの物しか無ければ材料にならない")

	stick(80)
	got, found := query.FindStackInInventory(world, "wooden_stick")
	assert.True(t, found)
	assert.Equal(t, 80, world.Components.Charge.Get(got).Current, "新品を返す")
}

func TestFindInBackpack_持ち主のバックパックの物だけを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	owner := world.ECS.NewEntity()
	other := world.ECS.NewEntity()
	place := func(id string, add func(e ecs.Entity)) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		add(e)
		return e
	}
	place("empty_battery", func(e ecs.Entity) {
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: other})
	})
	place("empty_battery", func(e ecs.Entity) {
		world.Components.LocationEquipped.Add(e, &gc.LocationEquipped{Owner: owner, EquipmentSlot: gc.SlotHand1})
	})
	mine := place("empty_battery", func(e ecs.Entity) {
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: owner})
	})

	got, ok := query.FindInBackpack(world, owner, "empty_battery")
	require.True(t, ok)
	assert.Equal(t, mine, got, "ほかの者の物と装備中の物は返さない")

	_, ok = query.FindInBackpack(world, owner, "battery")
	assert.False(t, ok, "持っていない物")
}
