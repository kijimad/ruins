package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestActiveFilter_退避中を除外し追加除外も効く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	active := world.ECS.NewEntity()
	world.Components.GridElement.Add(active, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})

	suspended := world.ECS.NewEntity()
	world.Components.GridElement.Add(suspended, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 2, Y: 2}})
	world.Components.Suspended.Add(suspended, &gc.Suspended{})

	tile := world.ECS.NewEntity()
	world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}})
	world.Components.Tile.Add(tile, &gc.Tile{})

	// Suspended は除外される。追加除外なし
	var got []ecs.Entity
	q := query.ActiveFilter1[gc.GridElement](world).Query()
	for q.Next() {
		got = append(got, q.Entity())
	}
	assert.Contains(t, got, active, "現ステージのエンティティは含まれる")
	assert.NotContains(t, got, suspended, "Suspended は除外される")
	assert.Contains(t, got, tile, "追加除外なしなら Tile も含まれる")

	// 追加除外 Tile を Without で重ねても効く
	var got2 []ecs.Entity
	q2 := query.ActiveFilter1[gc.GridElement](world).Without(ecs.C[gc.Tile]()).Query()
	for q2.Next() {
		got2 = append(got2, q2.Entity())
	}
	assert.Contains(t, got2, active, "追加除外があっても現ステージは含まれる")
	assert.NotContains(t, got2, suspended, "Suspended は依然除外される")
	assert.NotContains(t, got2, tile, "追加除外の Tile は外れる")
}
