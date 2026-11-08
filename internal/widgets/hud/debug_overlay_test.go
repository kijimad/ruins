package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDebugOverlay_既定で有効にする(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)

	overlay := NewDebugOverlay(res.Text.SmallFace)

	assert.True(t, overlay.enabled)
}

func TestDebugOverlay_Update_パニックしない(t *testing.T) {
	t.Parallel()
	overlay := &DebugOverlay{enabled: true}

	overlay.Update(w.World{})
}

func TestDebugOverlay_Draw_無効なら画面に触れない(t *testing.T) {
	t.Parallel()
	overlay := &DebugOverlay{enabled: false}

	// screen が nil でも早期に戻るのでパニックしない
	overlay.Draw(nil, DebugOverlayData{Enabled: true})
}

func TestDebugOverlay_Draw_データが無効なら画面に触れない(t *testing.T) {
	t.Parallel()
	overlay := &DebugOverlay{enabled: true}

	overlay.Draw(nil, DebugOverlayData{Enabled: false})
}

func TestDebugOverlay_Draw_有効ならAI状態と視界円とHPを描く(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	overlay := NewDebugOverlay(res.Text.SmallFace)
	screen := ebiten.NewImage(100, 100)

	overlay.Draw(screen, DebugOverlayData{
		Enabled: true,
		AIStates: []AIStateInfo{
			{Screen: consts.Coord[consts.ScreenPixel]{X: 50, Y: 50}, StateText: "Idle"},
		},
		VisionRanges: []VisionRangeInfo{
			{Screen: consts.Coord[consts.ScreenPixel]{X: 50, Y: 50}, ScaledRadius: 10},
		},
		HPDisplays: []HPDisplayInfo{
			{Screen: consts.Coord[consts.ScreenPixel]{X: 50, Y: 50}, CurrentHP: 5, MaxHP: 10},
		},
	})

	_, _, _, a := screen.At(50, 50).RGBA()
	assert.NotZero(t, a, "視界円の中心を塗る")
}
