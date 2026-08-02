package menuscreen_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/stretchr/testify/assert"
)

func TestWindowCursorReducer_上下で循環する(t *testing.T) {
	t.Parallel()
	reduce := menuscreen.WindowCursorReducer(3)
	assert.Equal(t, 1, reduce(0, inputmapper.ActionWindowDown), "下で次へ")
	assert.Equal(t, 2, reduce(1, inputmapper.ActionWindowDown))
	assert.Equal(t, 0, reduce(2, inputmapper.ActionWindowDown), "末尾の下は先頭へ循環")
	assert.Equal(t, 2, reduce(0, inputmapper.ActionWindowUp), "先頭の上は末尾へ循環")
	assert.Equal(t, 0, reduce(1, inputmapper.ActionWindowUp))
}

func TestWindowCursorReducer_選択肢が無いときは0に留まる(t *testing.T) {
	t.Parallel()
	reduce := menuscreen.WindowCursorReducer(0)
	assert.Equal(t, 0, reduce(0, inputmapper.ActionWindowDown))
	assert.Equal(t, 0, reduce(0, inputmapper.ActionWindowUp))
}

func TestWindowCursorReducer_無関係なアクションでは動かない(t *testing.T) {
	t.Parallel()
	reduce := menuscreen.WindowCursorReducer(3)
	assert.Equal(t, 1, reduce(1, inputmapper.ActionMenuSelect), "決定ではカーソルは動かない")
	assert.Equal(t, 2, reduce(2, inputmapper.ActionWindowConfirm))
}
