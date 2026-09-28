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

// TestCubeInfoItems_展開サイズと総重量と燃料を出す はキューブ情報の基本タブが展開サイズ・総重量・燃料の
// 3行を出すことを検証する。展開サイズはモジュール未装着の基準 5x5。
func TestCubeInfoItems_展開サイズと総重量と燃料を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	items := cubeInfoItems(world, cube)
	got := make(map[string]string, len(items))
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.Label
		got[it.Label] = it.Value
	}
	assert.Equal(t, []string{"Deploy size", "Total weight", "Fuel"}, labels, "展開サイズ・総重量・燃料の3行")
	assert.Equal(t, "5x5", got["Deploy size"], "未装着は基準 5x5")
}
