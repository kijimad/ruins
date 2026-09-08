package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCubeMenuState_4項目を並べる はキューブメニューが収納・オークション・情報・閉じるの
// 4項目を並べることを検証する。
func TestNewCubeMenuState_4項目を並べる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	st, err := NewCubeMenuState(cube)
	require.NoError(t, err)
	menu, ok := st.(*ChoiceMenuState)
	require.True(t, ok, "キューブメニューは ChoiceMenu ベース")

	_, choices := menu.provide(world)
	labels := make([]string, len(choices))
	for i, c := range choices {
		labels[i] = c.Label
	}
	want := []string{
		query.T(world, "Storage"),
		query.T(world, "Auction"),
		query.T(world, "Cube info"),
		query.T(world, "Close"),
	}
	assert.Equal(t, want, labels, "収納・オークション・キューブ情報・閉じるを順に並べる")
}
