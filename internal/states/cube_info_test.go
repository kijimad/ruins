package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCubeInfoState_基本タブを1つ持つ はキューブ情報メニューがいまは基本タブ1つを持つことを検証する。
// タブは後で増やす予定なので、1つであることを固定しておく。
func TestCubeInfoState_基本タブを1つ持つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	s, err := NewCubeInfoState(cube)
	require.NoError(t, err)
	st, ok := s.(*CubeInfoState)
	require.True(t, ok, "キューブ情報メニューは CubeInfoState")

	props, err := st.Fetch(world)
	require.NoError(t, err)
	require.Len(t, props.Tabs, 1, "いまはタブ1つ")
	assert.Equal(t, "Basic", props.Tabs[0].Label, "基本タブ")
}

// TestCubeInfoItems_総重量と燃料を出す はキューブ情報の基本タブが総重量と燃料の2行を出すことを検証する。
func TestCubeInfoItems_総重量と燃料を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	items := cubeInfoItems(world, cube)
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.Label
	}
	assert.Equal(t, []string{"Total weight", "Fuel"}, labels, "総重量と燃料の2行")
}
