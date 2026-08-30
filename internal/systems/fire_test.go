package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addFuelToStorage は火の収納へ燃料アイテムを1つ足す。
// firstFuelInStorage は SortEntities を通すため燃料には Name が要る。実アイテムは必ず Name を持つ
func addFuelToStorage(world w.World, fire ecs.Entity, name string, heatContent int) ecs.Entity {
	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: name})
	world.Components.Fuel.Add(item, &gc.Fuel{HeatContent: heatContent})
	world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: fire})
	return item
}

func TestFireSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("残量を毎ターン減らす", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 3})

		require.NoError(t, sys.Update(world))
		assert.Equal(t, 2, world.Components.Burning.Get(fire).Remaining)

		require.NoError(t, sys.Update(world))
		assert.Equal(t, 1, world.Components.Burning.Get(fire).Remaining)
	})

	t.Run("尽きたら収納の次の燃料へ移り効率を掛けて燃やす", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})
		fuel := addFuelToStorage(world, fire, "firewood", 10)

		// 残量1が0になり、収納の次の燃料へ移る。地面直の効率50%で 10*50/100 = 5
		require.NoError(t, sys.Update(world))
		assert.Equal(t, 5, world.Components.Burning.Get(fire).Remaining)
		assert.False(t, world.ECS.Alive(fuel), "燃やし始めた燃料は収納から取り除かれる")
		assert.True(t, world.Components.Burning.Has(fire), "次の燃料があるので火は続く")
	})

	t.Run("収納が空で残量も尽きると鎮火する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})

		// 残量1が0になり、次の燃料が無いので Burning が外れる
		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.Burning.Has(fire), "燃料が無ければ鎮火する")
	})

	t.Run("収納の燃料を上から順に食い尽くして最後に鎮火する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})
		// 効率50%で残量2ずつになる燃料を2つ。名前順で先に "a_log" が燃える
		addFuelToStorage(world, fire, "a_log", 4)
		addFuelToStorage(world, fire, "b_log", 4)

		// 1個目へ移る: 4*50/100 = 2
		require.NoError(t, sys.Update(world))
		assert.Equal(t, 2, world.Components.Burning.Get(fire).Remaining)

		// 2ターン燃やして0にし、2個目へ移る
		require.NoError(t, sys.Update(world))
		require.NoError(t, sys.Update(world))
		assert.Equal(t, 2, world.Components.Burning.Get(fire).Remaining)

		// 2個目も燃やし尽くし、収納が空なので鎮火する
		require.NoError(t, sys.Update(world))
		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.Burning.Has(fire), "全て燃やし尽くすと鎮火する")
	})
}
