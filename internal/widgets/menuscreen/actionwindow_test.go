package menuscreen

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActionWindow_初期状態は非表示(t *testing.T) {
	t.Parallel()
	a := NewActionWindow(func(_ w.World) (string, []Action, bool) { return "", nil, false })

	assert.False(t, a.Active())
}

func TestActionWindowOpen_先頭選択で表示状態にする(t *testing.T) {
	t.Parallel()
	a := NewActionWindow(func(_ w.World) (string, []Action, bool) { return "", nil, false })
	a.index = 3

	a.Open()

	assert.True(t, a.Active())
	assert.Equal(t, 0, a.index)
}

func TestActionWindowHandleInput_非表示のときは何もしない(t *testing.T) {
	t.Parallel()
	called := false
	a := NewActionWindow(func(_ w.World) (string, []Action, bool) {
		called = true
		return "", nil, false
	})

	err := a.HandleInput(w.World{})

	require.NoError(t, err)
	assert.False(t, called, "非表示のときprovideは呼ばれない")
	assert.False(t, a.Active())
}

func TestActionWindowWindow_対象が無ければnilを返す(t *testing.T) {
	t.Parallel()
	a := NewActionWindow(func(_ w.World) (string, []Action, bool) { return "", nil, false })

	got := a.Window(w.World{}, image.Rect(0, 0, 100, 100))

	assert.Nil(t, got)
}

//nolint:paralleltest // ebitenui内部のrace conditionのためt.Parallel()を使用しない
func TestActionWindowWindow_見出しと選択肢ラベルを表示する(t *testing.T) {
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	a := NewActionWindow(func(_ w.World) (string, []Action, bool) {
		return "操作", []Action{{Label: "つかう"}, {Label: "すてる"}}, true
	})

	win := a.Window(world, image.Rect(0, 0, 300, 200))

	require.NotNil(t, win)
	assert.Contains(t, collectLabels(win.TitleBar), "操作")
	contentLabels := collectLabels(win.Contents)
	assert.Contains(t, contentLabels, "つかう")
	assert.Contains(t, contentLabels, "すてる")
}
