package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedKeyboardInput(t *testing.T) {
	t.Parallel()

	keyboard1 := GetSharedKeyboardInput()
	keyboard2 := GetSharedKeyboardInput()

	assert.Same(t, keyboard1, keyboard2, "GetSharedKeyboardInput()は同一インスタンスを返す")
	require.NotNil(t, keyboard1)
}

func TestMockKeyboardInput_EnterFunction(t *testing.T) {
	t.Parallel()

	mock := NewMockKeyboardInput()

	mock.SetKeyPressed(ebiten.KeyEnter, true)
	assert.False(t, mock.IsEnterJustPressedOnce(), "押下状態のみでは検出されない")

	mock.SetKeyPressed(ebiten.KeyEnter, false)
	assert.True(t, mock.IsEnterJustPressedOnce(), "押下-押上のワンセットで検出される")

	assert.False(t, mock.IsEnterJustPressedOnce(), "押上状態での連続検出は発生しない")

	mock.Reset()
	assert.False(t, mock.IsKeyPressed(ebiten.KeyEnter), "Reset()後にキー状態がクリアされる")
}

func TestMockKeyboardInput_PressReleaseSequence(t *testing.T) {
	t.Parallel()

	mock := NewMockKeyboardInput()

	mock.SimulateEnterPressRelease()
	assert.True(t, mock.IsEnterJustPressedOnce(), "初回の押下-押上セットが検出される")

	mock.SimulateEnterPressRelease()
	assert.True(t, mock.IsEnterJustPressedOnce(), "2回目の押下-押上セットが検出される")

	mock.SetKeyPressed(ebiten.KeyEnter, true)
	assert.False(t, mock.IsEnterJustPressedOnce(), "押下のみでは検出されない")
	assert.False(t, mock.IsEnterJustPressedOnce(), "押下継続中は検出されない")

	mock.SetKeyPressed(ebiten.KeyEnter, false)
	assert.True(t, mock.IsEnterJustPressedOnce(), "押上時に検出される")
}

func TestMockKeyboardInput_BasicKeys(t *testing.T) {
	t.Parallel()

	mock := NewMockKeyboardInput()

	assert.False(t, mock.IsKeyPressed(ebiten.KeyA))
	mock.SetKeyPressed(ebiten.KeyA, true)
	assert.True(t, mock.IsKeyPressed(ebiten.KeyA))

	assert.False(t, mock.IsKeyJustPressed(ebiten.KeyB))
	mock.SetKeyJustPressed(ebiten.KeyB, true)
	assert.True(t, mock.IsKeyJustPressed(ebiten.KeyB))

	assert.False(t, mock.IsKeyPressedWithRepeat(ebiten.KeyC))
	mock.SetKeyPressedWithRepeat(ebiten.KeyC, true)
	assert.True(t, mock.IsKeyPressedWithRepeat(ebiten.KeyC))

	mock.Reset()
	assert.False(t, mock.IsKeyPressed(ebiten.KeyA))
	assert.False(t, mock.IsKeyJustPressed(ebiten.KeyB))
	assert.False(t, mock.IsKeyPressedWithRepeat(ebiten.KeyC))
}

func TestMockKeyboardInput_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ KeyboardInput = NewMockKeyboardInput()
}
