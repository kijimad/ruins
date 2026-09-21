package states

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCubeMenuState_圧縮中は展開を先頭に5項目を並べる(t *testing.T) {
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
		query.T(world, "Fuel"),
		query.T(world, "Cube info"),
		query.T(world, "Close"),
	}
	assert.Equal(t, want, labels, "圧縮中は展開・燃料・キューブ情報・閉じるを順に並べる")
}

func TestNewCubeMenuState_展開中は先頭が圧縮になる(t *testing.T) {
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
	assert.Equal(t, query.T(world, "Compress"), choices[0].Label, "展開中は先頭が圧縮")
}

func TestDeployChoice_展開できないときログを出しメニューを閉じる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	// 野営内(6,5)に prop を置いて展開を塞ぐ
	_, err = lifecycle.SpawnProp(world, "grass", 6, 5)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	trans, err := deployChoice(world, cube).Run(world)
	require.NoError(t, err)

	assert.False(t, world.Components.Deployed.Has(cube), "展開できないのでマーカーは付かない")
	assert.Equal(t, es.TransPop, trans.Type, "成否によらずメニューは閉じる")

	var logged bool
	for _, e := range query.GetGameLog(world).GetRecentEntries(5) {
		if strings.Contains(e.Text(), query.T(world, "Not enough open space to deploy.")) {
			logged = true
		}
	}
	assert.True(t, logged, "空き不足をゲームログに出す")
}
