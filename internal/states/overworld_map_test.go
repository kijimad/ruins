package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlyphColor_全ての種別記号に色が割り当てられている(t *testing.T) {
	t.Parallel()

	fallback := glyphColor('\x00') // 未知の文字の色
	for _, g := range overworld.PlaceGlyphs() {
		assert.NotEqualf(t, fallback, glyphColor(g.Label), "地物 %s(%c) に固有色がある", g.Name, g.Label)
	}
	for _, g := range overworld.FacilityGlyphs() {
		assert.NotEqualf(t, fallback, glyphColor(g.Label), "施設 %s(%c) に固有色がある", g.Name, g.Label)
	}
}

func TestGlyphColor_未知の文字は灰色のフォールバック(t *testing.T) {
	t.Parallel()

	assert.Equal(t, glyphColor('\x00'), glyphColor('Z'), "未知の文字は同じフォールバック色になる")
}

// TestOverworldMapState_キューブのチャンク位置を出す は大域地図にキューブのチャンク位置が
// マーカーとして載ることを検証する。現在地と重なるキューブは隠すので、プレイヤーを別チャンクへ寄せる。
func TestOverworldMapState_キューブのチャンク位置を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world)) // プレイヤー付近の中央チャンクにキューブを1体スポーンする

	// プレイヤーを中央のキューブと別チャンクへ動かす。同じチャンクだとキューブは向きポインタに隠れる
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 5

	st := &OverworldMapState{}
	require.NoError(t, st.OnStart(world))
	assert.NotEmpty(t, st.view.CubeCells, "現在地と別チャンクのキューブは大域地図に載る")
}
