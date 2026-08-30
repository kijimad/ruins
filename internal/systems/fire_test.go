package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFireSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("残量を毎ターン減らす", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 3})

		require.NoError(t, sys.Update(world))
		assert.Equal(t, consts.Turn(2), world.Components.Burning.Get(fire).Remaining)

		require.NoError(t, sys.Update(world))
		assert.Equal(t, consts.Turn(1), world.Components.Burning.Get(fire).Remaining)
	})

	t.Run("残量が尽きると鎮火して火が消える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})

		// 残量1が0になり Burning が外れ、火のエンティティは Dead になって除去される
		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.Burning.Has(fire), "残量が尽きれば鎮火する")
		assert.True(t, world.Components.Dead.Has(fire), "燃え尽きた火は Dead になり dead_cleanup が消す")
	})

	t.Run("燃え尽きると自分の HeatSource と LightSource を外す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		// 火は熱源と光源を持つ。暖房も灯りも component だけで決まるので、燃え尽きたら火の側で両方落とす
		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})
		world.Components.HeatSource.Add(fire, &gc.HeatSource{Radius: 2, Warmth: 0.75})
		world.Components.LightSource.Add(fire, &gc.LightSource{Enabled: true, Radius: 5})

		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.HeatSource.Has(fire), "燃え尽きた火は熱源を外して暖房を止める")
		assert.False(t, world.Components.LightSource.Has(fire), "燃え尽きた火は光源を外して灯りを止める")
	})

	t.Run("燃え尽きると火のあった場所に灰が残る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		coord := consts.Coord[consts.Tile]{X: 4, Y: 7}
		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})
		world.Components.GridElement.Add(fire, &gc.GridElement{Coord: coord})

		require.NoError(t, sys.Update(world))
		assert.True(t, world.Components.Dead.Has(fire), "燃え尽きた火は Dead になる")

		// 火のあった座標に灰アイテムが1つ落ちている
		var ashesAt int
		q := query.ActiveFilter2[gc.GridElement, gc.RawID](world).Query()
		for q.Next() {
			e := q.Entity()
			if world.Components.RawID.Get(e).ID == "ashes" &&
				world.Components.GridElement.Get(e).Coord == coord {
				ashesAt++
			}
		}
		assert.Equal(t, 1, ashesAt, "燃え尽きた跡には灰が1つ残るべき")
	})

	t.Run("座標を持たない火は灰を残さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})

		require.NoError(t, sys.Update(world))

		var ashesCount int
		q := query.ActiveFilter1[gc.RawID](world).Query()
		for q.Next() {
			if world.Components.RawID.Get(q.Entity()).ID == "ashes" {
				ashesCount++
			}
		}
		assert.Equal(t, 0, ashesCount, "座標がなければ灰の置き場所がないので残さない")
	})

	t.Run("残量ぶんだけ燃えて最後に鎮火する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 3})

		for range 2 {
			require.NoError(t, sys.Update(world))
			assert.True(t, world.Components.Burning.Has(fire))
		}
		// 3ターン目で残量0になり鎮火する
		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.Burning.Has(fire))
	})
}
