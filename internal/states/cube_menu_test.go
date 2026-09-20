package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCubeMenuState_収納中は展開を先頭に5項目を並べる(t *testing.T) {
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
		query.T(world, "Deploy"),
		query.T(world, "Storage"),
		query.T(world, "Auction"),
		query.T(world, "Cube info"),
		query.T(world, "Close"),
	}
	assert.Equal(t, want, labels, "収納中は展開・収納・オークション・キューブ情報・閉じるを順に並べる")
}

func TestNewCubeMenuState_展開中は先頭が収納になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})

	st, err := NewCubeMenuState(cube)
	require.NoError(t, err)
	menu, ok := st.(*ChoiceMenuState)
	require.True(t, ok)

	_, choices := menu.provide(world)
	assert.Equal(t, query.T(world, "Stow"), choices[0].Label, "展開中は先頭が収納")
}
