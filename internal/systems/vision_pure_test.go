package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVisionSystem_NewAndString はコンストラクタと名前を検証する。
func TestVisionSystem_NewAndString(t *testing.T) {
	t.Parallel()

	sys := NewVisionSystem()
	require.NotNil(t, sys)
	assert.Equal(t, "VisionSystem", sys.String())
}

// TestVisionSystem_Update_NoPlayer はプレイヤー不在時に何もせず nil を返すことを検証する。
// InitTestWorld はプレイヤーもダンジョンも持たないため、Update は最初の早期 return を通る。
func TestVisionSystem_Update_NoPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	sys := NewVisionSystem()

	require.NoError(t, sys.Update(world))
}

// TestTileRenderAt は描画情報の取得を検証する。
// map にあればその状態を返し、無ければ未探索の番兵 TileRenderUnexplored を返す。
func TestTileRenderAt(t *testing.T) {
	t.Parallel()

	present := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 2}}
	m := map[gc.GridElement]TileRenderInfo{
		present: TileRenderVisible{Darkness: 0.2},
	}

	// 在れば格納された状態を返す
	_, ok := tileRenderAt(m, present).(TileRenderVisible)
	assert.True(t, ok, "格納済みタイルは TileRenderVisible")

	// 不在は未探索の番兵
	absent := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}}
	_, ok = tileRenderAt(m, absent).(TileRenderUnexplored)
	assert.True(t, ok, "未格納タイルは TileRenderUnexplored")
}
