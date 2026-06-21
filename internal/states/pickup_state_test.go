package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickupState_StateConfig(t *testing.T) {
	t.Parallel()

	state := &PickupState{}
	config := state.StateConfig()
	assert.False(t, config.BlurBackground)
}

func TestPickupState_OnStart(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 7, "Ash")
	require.NoError(t, err)

	state := &PickupState{}
	err = state.OnStart(world)
	require.NoError(t, err)

	assert.Equal(t, consts.Tile(5), state.cursor.X)
	assert.Equal(t, consts.Tile(7), state.cursor.Y)
	assert.Equal(t, consts.Tile(5), state.playerPos.X)
	assert.Equal(t, consts.Tile(7), state.playerPos.Y)
}

func TestPickupState_moveCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dx, dy    int
		expectedX consts.Tile
		expectedY consts.Tile
	}{
		{"上", 0, -1, 10, 9},
		{"下", 0, 1, 10, 11},
		{"左", -1, 0, 9, 10},
		{"右", 1, 0, 11, 10},
		{"左上", -1, -1, 9, 9},
		{"右上", 1, -1, 11, 9},
		{"左下", -1, 1, 9, 11},
		{"右下", 1, 1, 11, 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := &PickupState{
				cursor:    consts.Coord[consts.Tile]{X: 10, Y: 10},
				playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
			}
			moveCursorAdjacent(&state.cursor, state.playerPos, tt.dx, tt.dy)
			assert.Equal(t, tt.expectedX, state.cursor.X)
			assert.Equal(t, tt.expectedY, state.cursor.Y)
		})
	}
}

func TestPickupState_moveCursor_ChebyshevConstraint(t *testing.T) {
	t.Parallel()

	t.Run("チェビシェフ距離1を超える移動はできない", func(t *testing.T) {
		t.Parallel()

		// カーソルが既に右上にある状態で、さらに右に移動しようとする
		state := &PickupState{
			cursor:    consts.Coord[consts.Tile]{X: 11, Y: 9},
			playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
		}
		moveCursorAdjacent(&state.cursor, state.playerPos, 1, 0) // X=12 はプレイヤーから距離2なので移動不可
		assert.Equal(t, consts.Tile(11), state.cursor.X)
		assert.Equal(t, consts.Tile(9), state.cursor.Y)
	})

	t.Run("チェビシェフ距離1以内なら移動できる", func(t *testing.T) {
		t.Parallel()

		state := &PickupState{
			cursor:    consts.Coord[consts.Tile]{X: 11, Y: 10},
			playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
		}
		moveCursorAdjacent(&state.cursor, state.playerPos, 0, -1) // X=11, Y=9 はプレイヤーから距離1なので移動可
		assert.Equal(t, consts.Tile(11), state.cursor.X)
		assert.Equal(t, consts.Tile(9), state.cursor.Y)
	})

	t.Run("足元から2マス先へは直接移動できない", func(t *testing.T) {
		t.Parallel()

		state := &PickupState{
			cursor:    consts.Coord[consts.Tile]{X: 10, Y: 9},
			playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
		}
		moveCursorAdjacent(&state.cursor, state.playerPos, 0, -1) // X=10, Y=8 はプレイヤーから距離2なので移動不可
		assert.Equal(t, consts.Tile(10), state.cursor.X)
		assert.Equal(t, consts.Tile(9), state.cursor.Y)
	})
}

func TestOffsetToLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dx, dy   int
		expected string
	}{
		{0, 0, "足元"},
		{0, -1, "上"},
		{0, 1, "下"},
		{-1, 0, "左"},
		{1, 0, "右"},
		{-1, -1, "左上"},
		{1, -1, "右上"},
		{-1, 1, "左下"},
		{1, 1, "右下"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, offsetToLabel(tt.dx, tt.dy))
		})
	}
}
