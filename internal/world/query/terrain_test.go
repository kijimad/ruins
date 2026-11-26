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

func TestForEachTerrainOrPropWithout_印を持たず壊れていない現ステージの地形と置物だけを回す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	at := func(x consts.Tile) *gc.GridElement {
		return &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x}}
	}

	wall := world.ECS.NewEntity()
	world.Components.GridElement.Add(wall, at(1))
	world.Components.Tile.Add(wall, &gc.Tile{})

	fence := world.ECS.NewEntity()
	world.Components.GridElement.Add(fence, at(2))
	world.Components.Prop.Add(fence, &gc.Prop{})

	floor := world.ECS.NewEntity()
	world.Components.GridElement.Add(floor, at(3))
	world.Components.Tile.Add(floor, &gc.Tile{})
	world.Components.Passable.Add(floor, &gc.Passable{})

	broken := world.ECS.NewEntity()
	world.Components.GridElement.Add(broken, at(4))
	world.Components.Prop.Add(broken, &gc.Prop{})
	world.Components.Dead.Add(broken, &gc.Dead{})

	suspended := world.ECS.NewEntity()
	world.Components.GridElement.Add(suspended, at(5))
	world.Components.Tile.Add(suspended, &gc.Tile{})
	world.Components.Suspended.Add(suspended, &gc.Suspended{})

	character := world.ECS.NewEntity()
	world.Components.GridElement.Add(character, at(6))

	var got []ecs.Entity
	query.ForEachTerrainOrPropWithout(world, func(e ecs.Entity) { got = append(got, e) }, ecs.C[gc.Passable]())

	assert.ElementsMatch(t, []ecs.Entity{wall, fence}, got)
}

func TestForEachTerrainOrPropWithout_印を複数渡すとどれか1つでも持てば除く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	at := func(x consts.Tile) *gc.GridElement {
		return &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x}}
	}

	wall := world.ECS.NewEntity()
	world.Components.GridElement.Add(wall, at(1))
	world.Components.Tile.Add(wall, &gc.Tile{})

	window := world.ECS.NewEntity()
	world.Components.GridElement.Add(window, at(2))
	world.Components.Tile.Add(window, &gc.Tile{})
	world.Components.SeeThrough.Add(window, &gc.SeeThrough{})

	thicket := world.ECS.NewEntity()
	world.Components.GridElement.Add(thicket, at(3))
	world.Components.Prop.Add(thicket, &gc.Prop{})
	world.Components.Passable.Add(thicket, &gc.Passable{})

	var got []ecs.Entity
	query.ForEachTerrainOrPropWithout(world, func(e ecs.Entity) { got = append(got, e) }, ecs.C[gc.SeeThrough](), ecs.C[gc.Passable]())

	assert.ElementsMatch(t, []ecs.Entity{wall}, got)
}

func TestIsStandingTerrainOrProp(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tile := world.ECS.NewEntity()
	world.Components.Tile.Add(tile, &gc.Tile{})
	prop := world.ECS.NewEntity()
	world.Components.Prop.Add(prop, &gc.Prop{})
	broken := world.ECS.NewEntity()
	world.Components.Prop.Add(broken, &gc.Prop{})
	world.Components.Dead.Add(broken, &gc.Dead{})
	item := world.ECS.NewEntity()
	world.Components.Item.Add(item, &gc.Item{})

	got := map[string]bool{
		"tile":   query.IsStandingTerrainOrProp(world, tile),
		"prop":   query.IsStandingTerrainOrProp(world, prop),
		"broken": query.IsStandingTerrainOrProp(world, broken),
		"item":   query.IsStandingTerrainOrProp(world, item),
	}
	assert.Equal(t, map[string]bool{"tile": true, "prop": true, "broken": false, "item": false}, got)
}
