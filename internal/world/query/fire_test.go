package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateBurnTurns_残量がそのままターン数になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	fire := world.ECS.NewEntity()
	world.Components.Burning.Add(fire, &gc.Burning{Remaining: 17})

	// 火は燃料を貯めず残量だけを持つ。残量は毎ターン1ずつ減るので残量=ターン数
	assert.Equal(t, consts.Turn(17), query.EstimateBurnTurns(world, fire))
}

func TestEstimateBurnTurns_燃えていなければ0(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	notFire := world.ECS.NewEntity()
	assert.Equal(t, consts.Turn(0), query.EstimateBurnTurns(world, notFire))
}

func TestFuelBurnTurns_熱量を効率で割り引く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	fire := world.ECS.NewEntity()
	world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})
	fuel := world.ECS.NewEntity()
	// 熱量は材質×重量から導く。WOOD 200/kg × 200g = 40
	world.Components.Material.Add(fuel, &gc.Material{Kind: rawschema.WOOD})
	world.Components.Weight.Add(fuel, &gc.Weight{Milligram: 200 * consts.MilligramPerGram})

	// 地面直の効率50%で 40*50/100 = 20 ターン
	assert.Equal(t, consts.Turn(20), query.FuelBurnTurns(world, fire, fuel))
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

func TestFireStarterOf_照らしていない物から残量の多い順に選ぶ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player := world.ECS.NewEntity()
	starter := func(charge int, equipped bool) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.FireStarter.Add(e, &gc.FireStarter{})
		if charge > 0 {
			world.Components.Charge.Add(e, &gc.Charge{Max: 600, Current: charge})
		}
		if equipped {
			world.Components.LocationEquipped.Add(e, &gc.LocationEquipped{Owner: player})
		} else {
			world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: player})
		}
		return e
	}

	equipped := starter(600, true)
	got, ok := query.FireStarterOf(world, player)
	require.True(t, ok)
	assert.Equal(t, equipped, got, "装備の物しか無ければそれを使う")

	low := starter(100, false)
	high := starter(400, false)
	got, _ = query.FireStarterOf(world, player)
	assert.Equal(t, high, got, "バックパックの物を装備の物より先に、残量の多い物から使う")

	world.ECS.RemoveEntity(high)
	got, _ = query.FireStarterOf(world, player)
	assert.Equal(t, low, got)

	endless := starter(0, true)
	got, _ = query.FireStarterOf(world, player)
	assert.Equal(t, endless, got, "残量を持たない物を最初に使う")
}
