package hud

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/messagelog"
	"github.com/stretchr/testify/assert"
)

func TestNewMessageArea_デフォルト設定で構築する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())

	area := NewMessageArea(world)

	assert.NotNil(t, area.widget, "ウィジェットは world から組み立てて保持する")
	assert.Equal(t, DefaultMessageAreaConfig, area.config, "デフォルト設定をそのまま使う")
	assert.Equal(t, Chrome{}, area.chrome, "意匠は状態を持たないゼロ値")
	assert.True(t, area.enabled, "初期状態は有効")
}

func TestMessageArea_Update_ガード条件はpanicしない(t *testing.T) {
	t.Parallel()

	t.Run("widget が nil でも panic しない", func(t *testing.T) {
		t.Parallel()
		area := &MessageArea{enabled: true, widget: nil}
		assert.NotPanics(t, func() { area.Update() })
	})

	t.Run("無効時は widget があっても呼ばない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithUI())
		area := NewMessageArea(world)
		area.enabled = false
		assert.NotPanics(t, func() { area.Update() })
	})
}

func TestMessageArea_Draw_無効またはwidgetなしなら何も描かない(t *testing.T) {
	t.Parallel()

	t.Run("無効なら描かない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithUI())
		area := NewMessageArea(world)
		area.enabled = false
		cv := &fakeCanvas{}

		area.Draw(cv, MessageData{ScreenDimensions: ScreenDimensions{Width: 960, Height: 720}})

		assert.Zero(t, cv.roundedFills, "パネルの背景を描かない")
		assert.Zero(t, cv.roundedStrokes, "パネルの枠を描かない")
	})

	t.Run("widget が nil なら描かない", func(t *testing.T) {
		t.Parallel()
		area := &MessageArea{enabled: true, widget: nil}
		cv := &fakeCanvas{}

		area.Draw(cv, MessageData{ScreenDimensions: ScreenDimensions{Width: 960, Height: 720}})

		assert.Zero(t, cv.roundedFills)
		assert.Zero(t, cv.roundedStrokes)
	})
}

func TestMessageArea_Draw_有効なら画面下部へパネルを1枚描く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	area := NewMessageArea(world)
	cv := &fakeCanvas{}

	area.Draw(cv, MessageData{ScreenDimensions: ScreenDimensions{Width: 960, Height: 720}})

	assert.Equal(t, 1, cv.roundedFills, "パネル背景は1回だけ敷く")
	assert.Equal(t, 1, cv.roundedStrokes, "パネル枠は1回だけ描く")
}

func TestMessagePanelWidget_Layout_矩形を保持する(t *testing.T) {
	t.Parallel()
	pw := &messagePanelWidget{}
	r := image.Rect(10, 20, 300, 80)

	pw.Layout(r)

	assert.Equal(t, r, pw.rect, "Layout で渡された矩形をそのまま保持する")
}

func TestMessagePanelWidget_Draw_背景と枠を1回ずつ描く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	widget := messagelog.NewWidget(messagelog.WidgetConfig{MaxLines: 5, LineHeight: LineH}, world)
	pw := &messagePanelWidget{chrome: Chrome{}, widget: widget, margin: DefaultMessageAreaConfig.LogAreaMargin}
	pw.Layout(image.Rect(0, 0, 400, 100))
	cv := &fakeCanvas{}

	pw.Draw(cv)

	assert.Equal(t, 1, cv.roundedFills, "背景パネルを1回敷く")
	assert.Equal(t, 1, cv.roundedStrokes, "パネル枠を1回描く")
}

func TestMessagePanelWidget_Draw_余白で潰れる極小矩形でもpanicしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	widget := messagelog.NewWidget(messagelog.WidgetConfig{MaxLines: 5, LineHeight: LineH}, world)
	// margin がはみ出すほど矩形が小さい。内側のログ本体の幅・高さが0以下になる境界
	pw := &messagePanelWidget{chrome: Chrome{}, widget: widget, margin: 50}
	pw.Layout(image.Rect(0, 0, 10, 10))
	cv := &fakeCanvas{}

	assert.NotPanics(t, func() { pw.Draw(cv) })
}
