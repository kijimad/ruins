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

func TestApplyCubeModuleChoice_収納から装着する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()

	m := world.ECS.NewEntity()
	world.Components.CubeModule.Add(m, &gc.CubeModule{RangeBonus: 1})
	world.Components.LocationInStorage.Add(m, &gc.LocationInStorage{Owner: cube})

	err := applyCubeModuleChoice(world, cube, cubeModuleChoice{entity: m}, nil)
	require.NoError(t, err)

	assert.True(t, world.Components.LocationCubeModule.Has(m), "装着で LocationCubeModule が付く")
	assert.False(t, world.Components.LocationInStorage.Has(m), "収納からは外れる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 3}, query.CubeDeployRange(world, cube), "装着で範囲が伸びる")
}

func TestApplyCubeModuleChoice_外して収納へ戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()

	m := world.ECS.NewEntity()
	world.Components.CubeModule.Add(m, &gc.CubeModule{RangeBonus: 1})
	world.Components.LocationCubeModule.Add(m, &gc.LocationCubeModule{Owner: cube})

	err := applyCubeModuleChoice(world, cube, cubeModuleChoice{remove: true}, &m)
	require.NoError(t, err)

	assert.True(t, world.Components.LocationInStorage.Has(m), "外すと収納へ戻る")
	assert.False(t, world.Components.LocationCubeModule.Has(m), "装着は外れる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 2, Y: 2}, query.CubeDeployRange(world, cube), "外すと基準へ戻る")
}
