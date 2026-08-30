package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

// addStorageFuel は fire の収納へ Fuel を持つアイテムを1つ足す
func addStorageFuel(world w.World, fire ecs.Entity, name string, heat int) {
	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: name})
	world.Components.Fuel.Add(item, &gc.Fuel{HeatContent: heat})
	world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: fire})
}

func TestEstimateBurnTurns_現残量と収納燃料を効率で割り引いて合算する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	fire := world.ECS.NewEntity()
	world.Components.Burning.Add(fire, &gc.Burning{Remaining: 3})
	// 効率50%で 20*50/100=10 と 8*50/100=4。現残量3と合わせて 3+10+4 = 17
	addStorageFuel(world, fire, "coal", 20)
	addStorageFuel(world, fire, "wood", 8)

	assert.Equal(t, 17, query.EstimateBurnTurns(world, fire))
}

func TestEstimateBurnTurns_燃えていなければ0(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	notFire := world.ECS.NewEntity()
	assert.Equal(t, 0, query.EstimateBurnTurns(world, notFire))
}

func TestHoldsFireStarter_バックパックと装備の火種を見て他人の物は除く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	assert.False(t, query.HoldsFireStarter(world, player), "初期状態では火種を持たない")

	// 他人のバックパックにある火種は数えない
	other := world.ECS.NewEntity()
	othersStarter := world.ECS.NewEntity()
	world.Components.FireStarter.Add(othersStarter, &gc.FireStarter{})
	world.Components.LocationInBackpack.Add(othersStarter, &gc.LocationInBackpack{Owner: other})
	assert.False(t, query.HoldsFireStarter(world, player), "他人の火種は数えない")

	// バックパックに火種を入れると true
	backpackStarter := world.ECS.NewEntity()
	world.Components.FireStarter.Add(backpackStarter, &gc.FireStarter{})
	world.Components.LocationInBackpack.Add(backpackStarter, &gc.LocationInBackpack{Owner: player})
	assert.True(t, query.HoldsFireStarter(world, player), "バックパックの火種を見る")
}

func TestHoldsFireStarter_装備した火種も所持とみなす(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	equippedStarter := world.ECS.NewEntity()
	world.Components.FireStarter.Add(equippedStarter, &gc.FireStarter{})
	world.Components.LocationEquipped.Add(equippedStarter, &gc.LocationEquipped{Owner: player})
	assert.True(t, query.HoldsFireStarter(world, player), "装備した火種も所持とみなす")
}
