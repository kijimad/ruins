package hud

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/messagelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageArea_既定設定で有効なエリアを作る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())

	area := NewMessageArea(world)

	require.NotNil(t, area.widget)
	assert.True(t, area.enabled)
	assert.Equal(t, DefaultMessageAreaConfig, area.config)
}

func TestMessageArea_Update_無効なら更新しない(t *testing.T) {
	t.Parallel()
	area := &MessageArea{enabled: false, widget: nil}

	// widget が nil でも無効なら触れずに戻る
	area.Update()
}

func TestMessageArea_Update_widgetが無ければ更新しない(t *testing.T) {
	t.Parallel()
	area := &MessageArea{enabled: true, widget: nil}

	// widget 未設定でも早期に戻ることを確認する
	area.Update()
}

func TestMessageArea_Update_有効なら委譲する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	area := NewMessageArea(world)

	// 実ウィジェットへ委譲してもパニックしない
	area.Update()
}

func TestMessageArea_Draw_無効なら何も描かない(t *testing.T) {
	t.Parallel()
	area := &MessageArea{enabled: false, widget: &messagelog.Widget{}}
	cv := &fakeCanvas{}

	area.Draw(cv, MessageData{ScreenDimensions: ScreenDimensions{Width: 960, Height: 720}})

	assert.Zero(t, cv.roundedFills)
	assert.Zero(t, cv.roundedStrokes)
}

func TestMessageArea_Draw_widgetが無ければ何も描かない(t *testing.T) {
	t.Parallel()
	area := &MessageArea{enabled: true, widget: nil}
	cv := &fakeCanvas{}

	area.Draw(cv, MessageData{ScreenDimensions: ScreenDimensions{Width: 960, Height: 720}})

	assert.Zero(t, cv.roundedFills)
}

func TestMessageArea_Draw_有効ならパネルを下部へ描く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	area := NewMessageArea(world)
	cv := &fakeCanvas{}
	screen := ScreenDimensions{Width: 960, Height: 720}

	area.Draw(cv, MessageData{ScreenDimensions: screen})

	assert.Equal(t, 1, cv.roundedFills, "背景パネルを1枚敷く")
	assert.Equal(t, 1, cv.roundedStrokes, "背景の枠を1本描く")
}

func TestMessagePanelWidget_Layout_矩形を保持する(t *testing.T) {
	t.Parallel()
	panel := &messagePanelWidget{}
	rect := image.Rect(1, 2, 3, 4)

	panel.Layout(rect)

	assert.Equal(t, rect, panel.rect)
}

func TestMessagePanelWidget_Children_子を持たない(t *testing.T) {
	t.Parallel()
	panel := &messagePanelWidget{}

	assert.Nil(t, panel.Children())
}

func TestMessagePanelWidget_Draw_内側余白ぶん縮めてログ本体を描く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	widget := messagelog.NewWidget(messagelog.WidgetConfig{MaxLines: 5, LineHeight: 20}, world)
	panel := &messagePanelWidget{chrome: Chrome{}, widget: widget, margin: 4}
	panel.Layout(image.Rect(0, 0, 100, 50))
	cv := &fakeCanvas{}

	panel.Draw(cv)

	assert.Equal(t, 1, cv.roundedFills, "背景枠を1枚敷く")
}
