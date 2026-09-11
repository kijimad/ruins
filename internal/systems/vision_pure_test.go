package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewVisionSystem_初期化してStringを返す はコンストラクタと名前を検証する。
func TestNewVisionSystem_初期化してStringを返す(t *testing.T) {
	t.Parallel()

	sys := NewVisionSystem()
	require.NotNil(t, sys)
	assert.Equal(t, "VisionSystem", sys.String())
}

// TestVisionSystem_Update_プレイヤー不在なら早期returnしnilを返す を検証する。
// InitTestWorld はプレイヤーもダンジョンも持たないため、Update は最初の早期 return を通る。
func TestVisionSystem_Update_プレイヤー不在なら早期returnしnilを返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	sys := NewVisionSystem()

	require.NoError(t, sys.Update(world))
}

// TestTileRenderAt_格納済みは同じ状態を返し不在は未探索番兵を返す を検証する。
func TestTileRenderAt_格納済みは同じ状態を返し不在は未探索番兵を返す(t *testing.T) {
	t.Parallel()

	present := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 2}}
	m := map[gc.GridElement]TileRenderInfo{
		present: TileRenderVisible{Darkness: 0.2},
	}

	// 在れば格納された状態を、同じ値のまま返す
	v, ok := tileRenderAt(m, present).(TileRenderVisible)
	require.True(t, ok, "格納済みタイルは TileRenderVisible")
	assert.Equal(t, VisibleDarkness(0.2), v.Darkness, "格納した値がそのまま返る")

	// 不在は未探索の番兵
	_, ok = tileRenderAt(m, gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}}).(TileRenderUnexplored)
	assert.True(t, ok, "未格納タイルは TileRenderUnexplored")
}
