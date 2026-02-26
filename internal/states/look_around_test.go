package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookAroundState_StateConfig(t *testing.T) {
	t.Parallel()

	state := &LookAroundState{}
	config := state.StateConfig()

	assert.False(t, config.BlurBackground, "LookAroundStateはブラーを適用しない")
}

func TestLookAroundState_OnStart(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Dungeon.Level.TileWidth = 50
	world.Resources.Dungeon.Level.TileHeight = 50

	playerEntity, err := worldhelper.SpawnPlayer(world, 5, 7, "セレスティン")
	require.NoError(t, err)

	state := &LookAroundState{}
	err = state.OnStart(world)
	require.NoError(t, err)

	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	assert.Equal(t, playerGrid.X, state.cursor.X, "カーソルX座標がプレイヤー位置と一致するべき")
	assert.Equal(t, playerGrid.Y, state.cursor.Y, "カーソルY座標がプレイヤー位置と一致するべき")
}

func TestLookAroundState_CursorMovement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		startX     int
		startY     int
		mapWidth   int
		mapHeight  int
		moveX      int
		moveY      int
		expectedX  int
		expectedY  int
		shouldMove bool
	}{
		{
			name:       "中央から右へ移動",
			startX:     5,
			startY:     5,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      1,
			moveY:      0,
			expectedX:  6,
			expectedY:  5,
			shouldMove: true,
		},
		{
			name:       "中央から左へ移動",
			startX:     5,
			startY:     5,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      -1,
			moveY:      0,
			expectedX:  4,
			expectedY:  5,
			shouldMove: true,
		},
		{
			name:       "中央から上へ移動",
			startX:     5,
			startY:     5,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      0,
			moveY:      -1,
			expectedX:  5,
			expectedY:  4,
			shouldMove: true,
		},
		{
			name:       "中央から下へ移動",
			startX:     5,
			startY:     5,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      0,
			moveY:      1,
			expectedX:  5,
			expectedY:  6,
			shouldMove: true,
		},
		{
			name:       "左端で左へ移動できない",
			startX:     0,
			startY:     5,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      -1,
			moveY:      0,
			expectedX:  0,
			expectedY:  5,
			shouldMove: false,
		},
		{
			name:       "上端で上へ移動できない",
			startX:     5,
			startY:     0,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      0,
			moveY:      -1,
			expectedX:  5,
			expectedY:  0,
			shouldMove: false,
		},
		{
			name:       "右端で右へ移動できない",
			startX:     19,
			startY:     5,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      1,
			moveY:      0,
			expectedX:  19,
			expectedY:  5,
			shouldMove: false,
		},
		{
			name:       "下端で下へ移動できない",
			startX:     5,
			startY:     19,
			mapWidth:   20,
			mapHeight:  20,
			moveX:      0,
			moveY:      1,
			expectedX:  5,
			expectedY:  19,
			shouldMove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)
			world.Resources.Dungeon.Level.TileWidth = consts.Tile(tt.mapWidth)
			world.Resources.Dungeon.Level.TileHeight = consts.Tile(tt.mapHeight)

			state := &LookAroundState{
				cursor: consts.Coord[consts.Tile]{X: consts.Tile(tt.startX), Y: consts.Tile(tt.startY)},
			}

			state.moveCursor(world, tt.moveX, tt.moveY)

			assert.Equal(t, consts.Tile(tt.expectedX), state.cursor.X, "カーソルX座標が期待値と異なる")
			assert.Equal(t, consts.Tile(tt.expectedY), state.cursor.Y, "カーソルY座標が期待値と異なる")
		})
	}
}
