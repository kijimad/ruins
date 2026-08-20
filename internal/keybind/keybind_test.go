package keybind

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

// TestReadInput は本番と再生で唯一分岐する点の挙動を固定する。world が供給源を持つなら
// そこから読み、持たないならキーボードから変換する
func TestReadInput(t *testing.T) {
	t.Parallel()

	t.Run("供給源があればそこから読む", func(t *testing.T) {
		t.Parallel()
		world := w.World{Resources: &resources.Resources{
			InputSource: func() (inputmapper.ActionID, bool) { return inputmapper.ActionMenuSelect, true },
		}}

		action, ok := ReadInput(world)

		assert.True(t, ok)
		assert.Equal(t, inputmapper.ActionMenuSelect, action)
	})

	t.Run("供給源が偽を返せば入力なしになる", func(t *testing.T) {
		t.Parallel()
		world := w.World{Resources: &resources.Resources{
			InputSource: func() (inputmapper.ActionID, bool) { return "", false },
		}}

		action, ok := ReadInput(world)

		assert.False(t, ok, "供給源が尽きたフレームは入力なしとして扱う")
		assert.Equal(t, inputmapper.ActionID(""), action)
	})

	t.Run("供給源が無ければキーボード経路になる", func(t *testing.T) {
		t.Parallel()
		world := w.World{Resources: &resources.Resources{}}

		action, ok := ReadInput(world)

		assert.False(t, ok, "本番経路。テストではキーが押されないので偽")
		assert.Equal(t, inputmapper.ActionID(""), action)
	})
}

// TestConvertKeys はキー→Action 変換を固定する。従来は各 state のキー直読みに散っていて
// テストが届かなかった対応を、モックキーボードでまとめて検証する
func TestConvertKeys(t *testing.T) {
	t.Parallel()

	t.Run("共通キーの対応", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			press func(ki *input.MockKeyboardInput)
			want  inputmapper.ActionID
		}{
			{"Escapeでキャンセル", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyEscape, true)
			}, inputmapper.ActionMenuCancel},
			{"左でタブ前へ", func(ki *input.MockKeyboardInput) {
				ki.SetKeyPressedWithRepeat(ebiten.KeyArrowLeft, true)
			}, inputmapper.ActionMenuTabPrev},
			{"右でタブ次へ", func(ki *input.MockKeyboardInput) {
				ki.SetKeyPressedWithRepeat(ebiten.KeyArrowRight, true)
			}, inputmapper.ActionMenuTabNext},
			{"上でカーソル上", func(ki *input.MockKeyboardInput) {
				ki.SetKeyPressedWithRepeat(ebiten.KeyArrowUp, true)
			}, inputmapper.ActionMenuUp},
			{"下でカーソル下", func(ki *input.MockKeyboardInput) {
				ki.SetKeyPressedWithRepeat(ebiten.KeyArrowDown, true)
			}, inputmapper.ActionMenuDown},
			{"Tabでタブ次へ", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyTab, true)
			}, inputmapper.ActionMenuTabNext},
			{"Shift+Tabでタブ前へ", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyTab, true)
				ki.SetKeyPressed(ebiten.KeyShift, true)
			}, inputmapper.ActionMenuTabPrev},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ki := input.NewMockKeyboardInput()
				tc.press(ki)

				action, ok := convertKeys(ki, nil)

				assert.True(t, ok)
				assert.Equal(t, tc.want, action)
			})
		}
	})

	t.Run("キーが無ければ入力なし", func(t *testing.T) {
		t.Parallel()
		action, ok := convertKeys(input.NewMockKeyboardInput(), nil)

		assert.False(t, ok)
		assert.Equal(t, inputmapper.ActionID(""), action)
	})

	t.Run("束縛表のShift条件", func(t *testing.T) {
		t.Parallel()
		bindings := []Binding{
			{Key: ebiten.KeyX, Shift: ShiftForbidden, Action: inputmapper.ActionOpenItemDetail},
			{Key: ebiten.KeyX, Shift: ShiftRequired, Action: inputmapper.ActionVerbExamine},
			{Key: ebiten.KeyD, Shift: ShiftAny, Action: inputmapper.ActionVerbPlace},
		}
		cases := []struct {
			name  string
			press func(ki *input.MockKeyboardInput)
			want  inputmapper.ActionID
		}{
			{"Shift無しのxは詳細", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyX, true)
			}, inputmapper.ActionOpenItemDetail},
			{"Shift+xは調べる", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyX, true)
				ki.SetKeyPressed(ebiten.KeyShift, true)
			}, inputmapper.ActionVerbExamine},
			{"ShiftAnyはShift無しでも効く", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyD, true)
			}, inputmapper.ActionVerbPlace},
			{"ShiftAnyはShift付きでも効く", func(ki *input.MockKeyboardInput) {
				ki.SetKeyJustPressed(ebiten.KeyD, true)
				ki.SetKeyPressed(ebiten.KeyShift, true)
			}, inputmapper.ActionVerbPlace},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ki := input.NewMockKeyboardInput()
				tc.press(ki)

				action, ok := convertKeys(ki, bindings)

				assert.True(t, ok)
				assert.Equal(t, tc.want, action)
			})
		}
	})

	t.Run("束縛表は共通キーより先に効く", func(t *testing.T) {
		t.Parallel()
		ki := input.NewMockKeyboardInput()
		ki.SetKeyJustPressed(ebiten.KeyEscape, true)
		bindings := []Binding{{Key: ebiten.KeyEscape, Shift: ShiftAny, Action: inputmapper.ActionCloseMenu}}

		action, ok := convertKeys(ki, bindings)

		assert.True(t, ok)
		assert.Equal(t, inputmapper.ActionCloseMenu, action, "同じキーなら束縛表が共通変換を上書きする")
	})
}
