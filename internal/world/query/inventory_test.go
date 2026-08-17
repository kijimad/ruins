package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

func TestFindStackableInInventory_名前が一致するバックパック内アイテムを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})
	world.Components.RawID.Add(item, &gc.RawID{ID: "回復薬"})

	got, found := query.FindStackableInInventory(world, "回復薬")
	assert.True(t, found)
	assert.Equal(t, item, got)
}

func TestFindStackableInInventory_名前が一致しなければ見つからない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

	_, found := query.FindStackableInInventory(world, "毒薬")
	assert.False(t, found)
}

func TestFindStackableInInventory_Stackableでなければ対象外(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

	_, found := query.FindStackableInInventory(world, "回復薬")
	assert.False(t, found)
}

func TestFindStackableInInventory_バックパック内でなければ対象外(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

	_, found := query.FindStackableInInventory(world, "回復薬")
	assert.False(t, found)
}

func TestFindAmmoInInventory_口径タグが一致するバックパック内弾薬を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	ammo := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(ammo, &gc.LocationInBackpack{Owner: owner})
	world.Components.Ammo.Add(ammo, &gc.Ammo{AmmoTag: oapi.N9mm})

	got, found := query.FindAmmoInInventory(world, oapi.N9mm)
	assert.True(t, found)
	assert.Equal(t, ammo, got)
}

func TestFindAmmoInInventory_口径タグが一致しなければ見つからない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	ammo := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(ammo, &gc.LocationInBackpack{Owner: owner})
	world.Components.Ammo.Add(ammo, &gc.Ammo{AmmoTag: oapi.N9mm})

	_, found := query.FindAmmoInInventory(world, oapi.Rifle)
	assert.False(t, found)
}

func TestFindAmmoInInventory_該当が無ければゼロ値エンティティを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	got, found := query.FindAmmoInInventory(world, oapi.Shell)
	assert.False(t, found)
	assert.Zero(t, got)
}
