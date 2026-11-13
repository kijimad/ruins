package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverworldMapState_キューブのチャンク位置を出す は大域地図にキューブのチャンク位置が
// マーカーとして載ることを検証する。現在地と重なるキューブは隠すので、プレイヤーを別チャンクへ寄せる。
func TestOverworldMapState_キューブのチャンク位置を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 24, 24, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world)) // プレイヤー付近の中央チャンクにキューブを1体スポーンする

	// プレイヤーを中央のキューブと別チャンクへ動かす。同じチャンクだとキューブは向きポインタに隠れる
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 5

	st := &OverworldMapState{}
	require.NoError(t, st.OnStart(world))
	assert.NotEmpty(t, st.view.CubeCells, "現在地と別チャンクのキューブは大域地図に載る")
}

func TestOverworldMapState_Update_ヘルプキーでキー一覧を開く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return inputmapper.ActionOpenKeyHelp, true }

	st := &OverworldMapState{}
	trans, err := st.Update(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransPush, trans.Type)
	require.Len(t, trans.NewStateFuncs, 1)
	pushed, err := trans.NewStateFuncs[0]()
	require.NoError(t, err)
	assert.IsType(t, &menuloop.KeyHelpState{}, pushed)
}
