package mapplanner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSelectorTestPlan は position_selector 検証用のマップを作る。allFloor が true なら全面床、
// false なら全面壁にする
func newSelectorTestPlan(allFloor bool) *MetaPlan {
	width, height := consts.Tile(5), consts.Tile(5)
	tiles := make([]oapi.Tile, int(width)*int(height))
	plan := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     tiles,
		RawMaster: CreateTestRawMaster(),
		RNG:       rand.New(rand.NewPCG(1, 2)),
	}
	tileName := "wall"
	if allFloor {
		tileName = "floor"
	}
	for i := range tiles {
		tiles[i] = plan.GetTile(tileName)
	}
	return plan
}

func TestOnMapSelector_スポーン可能なタイルがあれば座標を返す(t *testing.T) {
	t.Parallel()

	plan := newSelectorTestPlan(true)
	sel := onMapSelector(5)

	tx, ty, ok := sel(plan, w.World{})
	assert.True(t, ok)
	assert.True(t, tx >= 0 && tx < plan.Level.TileWidth)
	assert.True(t, ty >= 0 && ty < plan.Level.TileHeight)
}

func TestOnMapSelector_スポーン可能なタイルが無ければ試行回数内で失敗する(t *testing.T) {
	t.Parallel()

	plan := newSelectorTestPlan(false)
	sel := onMapSelector(3)

	tx, ty, ok := sel(plan, w.World{})
	assert.False(t, ok)
	assert.Equal(t, consts.Tile(0), tx)
	assert.Equal(t, consts.Tile(0), ty)
}

func TestNearSelector_中心座標がスポーン可能なら中心を返す(t *testing.T) {
	t.Parallel()

	plan := newSelectorTestPlan(true)
	room := gc.Rect{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 3, Y: 3}}
	// radius=0 なので候補座標は常に中心そのものになる
	sel := nearSelector(2, 2, 0, room, 1)

	tx, ty, ok := sel(plan, w.World{})
	assert.True(t, ok)
	assert.Equal(t, consts.Tile(2), tx)
	assert.Equal(t, consts.Tile(2), ty)
}

func TestFindPosition_最初に成功したセレクタの座標を返す(t *testing.T) {
	t.Parallel()

	plan := newSelectorTestPlan(true)
	alwaysFail := func(_ *MetaPlan, _ w.World) (consts.Tile, consts.Tile, bool) { return 0, 0, false }
	alwaysSucceed := func(_ *MetaPlan, _ w.World) (consts.Tile, consts.Tile, bool) { return 4, 3, true }

	tx, ty, err := findPosition(plan, w.World{}, alwaysFail, alwaysSucceed)
	require.NoError(t, err)
	assert.Equal(t, consts.Tile(4), tx)
	assert.Equal(t, consts.Tile(3), ty)
}

func TestFindPosition_全セレクタが失敗するとエラーを返す(t *testing.T) {
	t.Parallel()

	plan := newSelectorTestPlan(true)
	alwaysFail := func(_ *MetaPlan, _ w.World) (consts.Tile, consts.Tile, bool) { return 0, 0, false }

	_, _, err := findPosition(plan, w.World{}, alwaysFail, alwaysFail)
	assert.ErrorContains(t, err, "all 2 selectors failed")
}
