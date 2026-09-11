package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugOverlay_NewAndUpdate はコンストラクタと no-op の Update を検証する。
func TestDebugOverlay_NewAndUpdate(t *testing.T) {
	t.Parallel()

	overlay := NewDebugOverlay(nil)
	require.NotNil(t, overlay)

	world := testutil.InitTestWorld(t)
	assert.NotPanics(t, func() { overlay.Update(world) }, "Update は現状 no-op でパニックしない")
}

// TestDebugOverlay_Draw_無効なら何も描かない は data.Enabled が偽のとき早期 return することを検証する。
func TestDebugOverlay_Draw_無効なら何も描かない(t *testing.T) {
	t.Parallel()

	overlay := NewDebugOverlay(nil)
	screen := ebiten.NewImage(320, 240)

	assert.NotPanics(t, func() {
		overlay.Draw(screen, DebugOverlayData{Enabled: false})
	})
}

// TestDebugOverlay_Draw_状態と視界とHPを描く は有効時に AI状態の文字、視界円、HP文字を
// 実 screen へ描いても落ちないことを検証する。Draw は内部で EbitenCanvas を作るため実 face が要る。
func TestDebugOverlay_Draw_状態と視界とHPを描く(t *testing.T) {
	t.Parallel()

	res, err := loader.LoadUIResources()
	require.NoError(t, err)

	overlay := NewDebugOverlay(res.Text.BodyFace)
	screen := ebiten.NewImage(320, 240)

	at := consts.Coord[consts.ScreenPixel]{X: 100, Y: 100}
	data := DebugOverlayData{
		Enabled:          true,
		AIStates:         []AIStateInfo{{Screen: at, StateText: "PATROL"}},
		VisionRanges:     []VisionRangeInfo{{Screen: at, ScaledRadius: 48}},
		HPDisplays:       []HPDisplayInfo{{Screen: at, CurrentHP: 7, MaxHP: 10}},
		ScreenDimensions: ScreenDimensions{Width: 320, Height: 240},
	}

	assert.NotPanics(t, func() { overlay.Draw(screen, data) })
}
