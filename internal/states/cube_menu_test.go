package states

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// choiceByLabel はラベルで選択肢を1つ引く。無効フラグの検査に使う
func choiceByLabel(choices []Choice, label string) (Choice, bool) {
	for _, c := range choices {
		if c.Label == label {
			return c, true
		}
	}
	return Choice{}, false
}

func TestNewCubeMenuState_圧縮中は追従が有効で設備が無効(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.SoloAI.Get(cube).Movement = rawschema.Stationary

	st, err := NewCubeMenuState(cube)
	require.NoError(t, err)
	menu, ok := st.(*ChoiceMenuState)
	require.True(t, ok, "キューブメニューは ChoiceMenu ベース")

	_, choices := menu.provide(world)
	labels := make([]string, len(choices))
	for i, c := range choices {
		labels[i] = c.Label
	}
	// 状態で使えない項目も消さず無効表示で残すので、圧縮中も設備は並ぶ
	want := []string{
		query.T(world, "Deploy"),
		query.T(world, "Follow me"),
		query.T(world, "Fuel"),
		query.T(world, "Module"),
		query.T(world, "Facility"),
		query.T(world, "Info"),
		query.T(world, "Close"),
	}
	assert.Equal(t, want, labels)

	follow, _ := choiceByLabel(choices, query.T(world, "Follow me"))
	assert.False(t, follow.Disabled, "圧縮中は追従させられる")
	facility, _ := choiceByLabel(choices, query.T(world, "Facility"))
	assert.True(t, facility.Disabled, "圧縮中は展開空間が無いので設備は無効")
}

func TestNewCubeMenuState_展開中も追従を命じられ設備が有効(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.SoloAI.Get(cube).Movement = rawschema.Stationary
	world.Components.Deployed.Add(cube, &gc.Deployed{})

	st, err := NewCubeMenuState(cube)
	require.NoError(t, err)
	menu, ok := st.(*ChoiceMenuState)
	require.True(t, ok)

	_, choices := menu.provide(world)
	assert.Equal(t, query.T(world, "Compress"), choices[0].Label, "展開中は先頭が圧縮")

	follow, ok := choiceByLabel(choices, query.T(world, "Follow me"))
	require.True(t, ok)
	assert.False(t, follow.Disabled, "展開中も方針を切り替えられる")
	facility, ok := choiceByLabel(choices, query.T(world, "Facility"))
	require.True(t, ok, "展開中も設備は並ぶ")
	assert.False(t, facility.Disabled, "展開中は設備を装着できる")
}

func TestFollowChoice_追従と待機を切り替える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.SoloAI.Get(cube).Movement = rawschema.Stationary

	follow := followChoice(world, cube)
	assert.Equal(t, query.T(world, "Follow me"), follow.Label, "待機中は追従を命じる")
	trans, err := follow.Run(world)
	require.NoError(t, err)
	assert.Equal(t, rawschema.Follow, world.Components.SoloAI.Get(cube).Movement)
	assert.Equal(t, es.TransNone, trans.Type, "設定の切り替えなのでメニューは閉じない")

	stay := followChoice(world, cube)
	assert.Equal(t, query.T(world, "Stay here"), stay.Label, "追従中は待機を命じる")
	_, err = stay.Run(world)
	require.NoError(t, err)
	assert.Equal(t, rawschema.Stationary, world.Components.SoloAI.Get(cube).Movement)
}

func TestDeployChoice_展開しても追従の方針を保つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	_, err = deployChoice(world, cube).Run(world)
	require.NoError(t, err)

	assert.True(t, world.Components.Deployed.Has(cube))
	assert.Equal(t, rawschema.Follow, world.Components.SoloAI.Get(cube).Movement)
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
