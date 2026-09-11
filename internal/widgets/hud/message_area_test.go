package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageArea_作成と更新と描画が通る は、HUDメッセージエリアを作り、更新と描画が
// panic せず通ることを固定する。UIツリー構築とキャンバス描画は WithUI で window 無しに動く。
func TestMessageArea_作成と更新と描画が通る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())

	area := NewMessageArea(world)
	require.NotNil(t, area)
	assert.True(t, area.enabled, "作成直後は有効")

	assert.NotPanics(t, func() { area.Update() })

	cv := uicore.NewEbitenCanvas(ebiten.NewImage(800, 600))
	data := MessageData{
		Messages:         []string{"テスト"},
		ScreenDimensions: ScreenDimensions{Width: 800, Height: 600},
		Config:           DefaultMessageAreaConfig,
	}
	assert.NotPanics(t, func() { area.Draw(cv, data) })
}

// TestMessageArea_無効なら更新も描画もしない は、enabled が偽のとき更新・描画が
// 早期に戻り panic しないことを固定する。
func TestMessageArea_無効なら更新も描画もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())

	// 先にテスト用リソースと対象の状態を整えてからアクションする
	cv := uicore.NewEbitenCanvas(ebiten.NewImage(800, 600))
	data := MessageData{
		ScreenDimensions: ScreenDimensions{Width: 800, Height: 600},
		Config:           DefaultMessageAreaConfig,
	}
	area := NewMessageArea(world)
	area.enabled = false

	// 有効時と同じく、どちらで panic したか分かるよう Update と Draw を分けて包む
	assert.NotPanics(t, func() { area.Update() })
	assert.NotPanics(t, func() { area.Draw(cv, data) })
}
