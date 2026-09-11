package messagewindow

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newChoiceWindow は選択肢2つを持つメッセージウィンドウと、入力を差し替える供給源、
// 選ばれた選択肢のテキストを記録するポインタを返す。
func newChoiceWindow(t *testing.T) (*Window, *inputmapper.ActionID, *string) {
	t.Helper()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	var action inputmapper.ActionID
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return action, true }

	selected := ""
	msg := messagedata.NewSystemMessage("質問").
		WithChoice("はい", func(_ w.World) error { selected = "はい"; return nil }).
		WithChoice("いいえ", func(_ w.World) error { selected = "いいえ"; return nil })

	return NewWindow(world, msg), &action, &selected
}

// TestWindowUpdate_選択肢キャンセルで閉じる は、選択肢表示中のキャンセル入力で
// ウィンドウが閉じることを固定する。
func TestWindowUpdate_選択肢キャンセルで閉じる(t *testing.T) {
	t.Parallel()

	win, action, _ := newChoiceWindow(t)
	require.True(t, win.isOpen)

	*action = inputmapper.ActionMenuCancel
	require.NoError(t, win.Update())

	assert.False(t, win.isOpen, "キャンセルで閉じる")
}

// TestWindowUpdate_選択肢決定でActionが走る は、先頭の選択肢を決定するとその Action が
// 呼ばれることを固定する。
func TestWindowUpdate_選択肢決定でActionが走る(t *testing.T) {
	t.Parallel()

	win, action, selected := newChoiceWindow(t)

	*action = inputmapper.ActionMenuSelect
	require.NoError(t, win.Update())

	assert.Equal(t, "はい", *selected, "先頭の選択肢のActionが走る")
}

// TestWindowUpdate_下移動してから決定で次の選択肢を選ぶ は、下入力で選択が動き、
// その後の決定で2つ目の選択肢が選ばれることを固定する。
func TestWindowUpdate_下移動してから決定で次の選択肢を選ぶ(t *testing.T) {
	t.Parallel()

	win, action, selected := newChoiceWindow(t)

	*action = inputmapper.ActionMenuDown
	require.NoError(t, win.Update())
	assert.True(t, win.isOpen, "移動では閉じない")

	*action = inputmapper.ActionMenuSelect
	require.NoError(t, win.Update())

	assert.Equal(t, "いいえ", *selected, "下移動後の決定で2つ目を選ぶ")
}
