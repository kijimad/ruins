package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCubeModuleChoiceAt_装着済みは先頭が外す(t *testing.T) {
	t.Parallel()

	installed := ecs.Entity{}
	c0 := ecs.Entity{}
	props := CubeModuleSelectProps{Candidates: []ecs.Entity{c0}, Installed: &installed}

	got, ok := cubeModuleChoiceAt(props, 0)
	require.True(t, ok)
	assert.True(t, got.remove, "先頭は外す")

	got, ok = cubeModuleChoiceAt(props, 1)
	require.True(t, ok)
	assert.False(t, got.remove, "候補は外すの後ろへ詰まる")

	_, ok = cubeModuleChoiceAt(props, 2)
	assert.False(t, ok, "範囲外は false")
}

func TestCubeModuleChoiceAt_空きは候補だけ(t *testing.T) {
	t.Parallel()

	c0 := ecs.Entity{}
	props := CubeModuleSelectProps{Candidates: []ecs.Entity{c0}, Installed: nil}

	got, ok := cubeModuleChoiceAt(props, 0)
	require.True(t, ok)
	assert.False(t, got.remove, "空きスロットは先頭から候補")
}

func TestCubeModuleMenuFetch_スロット範囲外はエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()

	m := world.ECS.NewEntity()
	world.Components.CubeModule.Add(m, &gc.CubeModule{RangeBonus: 1})
	world.Components.LocationInstalled.Add(m, &gc.LocationInstalled{Owner: cube, Slot: 99})

	st := &CubeModuleMenuState{cube: cube}
	_, err := st.Fetch(world)
	require.Error(t, err, "範囲外スロットは握りつぶさず error で返す")
}

func TestApplyCubeModuleChoice_バックパックから装着する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})

	m := world.ECS.NewEntity()
	world.Components.CubeModule.Add(m, &gc.CubeModule{RangeBonus: 1})
	world.Components.LocationInBackpack.Add(m, &gc.LocationInBackpack{Owner: player})

	err := applyCubeModuleChoice(world, cube, 0, cubeModuleChoice{entity: m}, nil)
	require.NoError(t, err)

	assert.True(t, world.Components.LocationInstalled.Has(m), "装着で LocationInstalled が付く")
	assert.Equal(t, 0, world.Components.LocationInstalled.Get(m).Slot, "指定スロットに入る")
	assert.False(t, world.Components.LocationInBackpack.Has(m), "バックパックからは外れる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 3}, query.CubeDeployRange(world, cube), "装着で範囲が伸びる")
}

func TestApplyCubeModuleChoice_外してバックパックへ戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})

	m := world.ECS.NewEntity()
	world.Components.CubeModule.Add(m, &gc.CubeModule{RangeBonus: 1})
	world.Components.LocationInstalled.Add(m, &gc.LocationInstalled{Owner: cube})

	err := applyCubeModuleChoice(world, cube, 0, cubeModuleChoice{remove: true}, &m)
	require.NoError(t, err)

	assert.True(t, world.Components.LocationInBackpack.Has(m), "外すとバックパックへ戻る")
	assert.False(t, world.Components.LocationInstalled.Has(m), "装着は外れる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 2, Y: 2}, query.CubeDeployRange(world, cube), "外すと基準へ戻る")
}

func TestApplyCubeModuleChoice_スロットを外しても他は繰り上がらない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})

	backpackModule := func() ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.CubeModule.Add(e, &gc.CubeModule{RangeBonus: 1})
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: player})
		return e
	}

	// スロット0と1にモジュールを装着する
	m0 := backpackModule()
	require.NoError(t, applyCubeModuleChoice(world, cube, 0, cubeModuleChoice{entity: m0}, nil))
	m1 := backpackModule()
	require.NoError(t, applyCubeModuleChoice(world, cube, 1, cubeModuleChoice{entity: m1}, nil))

	// スロット0を外す。スロット1のモジュールはスロット1のまま繰り上がらない
	require.NoError(t, applyCubeModuleChoice(world, cube, 0, cubeModuleChoice{remove: true}, &m0))

	require.True(t, world.Components.LocationInstalled.Has(m1))
	assert.Equal(t, 1, world.Components.LocationInstalled.Get(m1).Slot, "スロット1のまま繰り上がらない")
}
