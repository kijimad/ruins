package menuinput

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

// TestReadMenuInput は本番と再生で唯一分岐する点の挙動を固定する。world が供給源を持つなら
// そこから読み、持たないならキーボードから変換する
func TestReadMenuInput(t *testing.T) {
	t.Parallel()

	t.Run("供給源があればそこから読む", func(t *testing.T) {
		t.Parallel()
		world := w.World{Resources: &resources.Resources{
			MenuInput: func() (inputmapper.ActionID, bool) { return inputmapper.ActionMenuSelect, true },
		}}

		action, ok := ReadMenuInput(world)

		assert.True(t, ok)
		assert.Equal(t, inputmapper.ActionMenuSelect, action)
	})

	t.Run("供給源が偽を返せば入力なしになる", func(t *testing.T) {
		t.Parallel()
		world := w.World{Resources: &resources.Resources{
			MenuInput: func() (inputmapper.ActionID, bool) { return "", false },
		}}

		action, ok := ReadMenuInput(world)

		assert.False(t, ok, "供給源が尽きたフレームは入力なしとして扱う")
		assert.Equal(t, inputmapper.ActionID(""), action)
	})

	t.Run("供給源が無ければキーボード経路になる", func(t *testing.T) {
		t.Parallel()
		world := w.World{Resources: &resources.Resources{}}

		action, ok := ReadMenuInput(world)

		assert.False(t, ok, "本番経路。テストではキーが押されないので偽")
		assert.Equal(t, inputmapper.ActionID(""), action)
	})
}
