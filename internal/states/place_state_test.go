package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceState_StateConfig(t *testing.T) {
	t.Parallel()

	state := &PlaceState{}
	config := state.StateConfig()
	assert.False(t, config.BlurBackground)
}

func TestPlaceState_OnStart(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 3, Y: 8}, "Ash")
	require.NoError(t, err)

	state := &PlaceState{}
	err = state.OnStart(world)
	require.NoError(t, err)

	assert.Equal(t, consts.Tile(3), state.playerPos.X)
	assert.Equal(t, consts.Tile(8), state.playerPos.Y)
	assert.Equal(t, placePhaseSelectItem, state.phase)
}

func TestPlaceState_moveCursor(t *testing.T) {
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

			state := &PlaceState{
				cursor:    consts.Coord[consts.Tile]{X: 10, Y: 10},
				playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
			}
			moveCursorAdjacent(&state.cursor, state.playerPos, tt.dx, tt.dy)
			assert.Equal(t, tt.expectedX, state.cursor.X)
			assert.Equal(t, tt.expectedY, state.cursor.Y)
		})
	}
}

func TestPlaceState_moveCursor_ChebyshevConstraint(t *testing.T) {
	t.Parallel()

	t.Run("チェビシェフ距離1を超える移動はできない", func(t *testing.T) {
		t.Parallel()

		state := &PlaceState{
			cursor:    consts.Coord[consts.Tile]{X: 11, Y: 10},
			playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
		}
		moveCursorAdjacent(&state.cursor, state.playerPos, 1, 0) // X=12 はプレイヤーから距離2なので移動不可
		assert.Equal(t, consts.Tile(11), state.cursor.X)
		assert.Equal(t, consts.Tile(10), state.cursor.Y)
	})

	t.Run("斜め端からさらに斜めに移動できない", func(t *testing.T) {
		t.Parallel()

		state := &PlaceState{
			cursor:    consts.Coord[consts.Tile]{X: 11, Y: 11},
			playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
		}
		moveCursorAdjacent(&state.cursor, state.playerPos, 1, 0) // X=12, Y=11 はプレイヤーから距離2なので移動不可
		assert.Equal(t, consts.Tile(11), state.cursor.X)
		assert.Equal(t, consts.Tile(11), state.cursor.Y)
	})

	t.Run("足元への移動は可能", func(t *testing.T) {
		t.Parallel()

		state := &PlaceState{
			cursor:    consts.Coord[consts.Tile]{X: 11, Y: 10},
			playerPos: consts.Coord[consts.Tile]{X: 10, Y: 10},
		}
		moveCursorAdjacent(&state.cursor, state.playerPos, -1, 0) // X=10, Y=10 はプレイヤー位置なので移動可
		assert.Equal(t, consts.Tile(10), state.cursor.X)
		assert.Equal(t, consts.Tile(10), state.cursor.Y)
	})
}
