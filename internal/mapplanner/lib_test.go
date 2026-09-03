package mapplanner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

func TestPlannerTypeByName_通常プランナーを名前で引ける(t *testing.T) {
	t.Parallel()

	pt, ok := PlannerTypeByName("Small Room")
	assert.True(t, ok)
	assert.Equal(t, PlannerTypeSmallRoom.Name, pt.Name)
}

func TestPlannerTypeByName_デバッグ用プランナーも名前で引ける(t *testing.T) {
	t.Parallel()

	pt, ok := PlannerTypeByName("Debug Town")
	assert.True(t, ok)
	assert.Equal(t, PlannerTypeDebugTown.Name, pt.Name)
}

func TestPlannerTypeByName_存在しない名前はfalseを返す(t *testing.T) {
	t.Parallel()

	pt, ok := PlannerTypeByName("存在しないプランナー")
	assert.False(t, ok)
	assert.Equal(t, PlannerType{}, pt)
}

// newRandomPositionNearTestPlan は randomPositionNear 検証用の5x5全壁マップを作る。
// 中心タイルだけを floor / wall で使い分けられるよう呼び出し側に委ねる
func newRandomPositionNearTestPlan(centerFloor bool, centerX, centerY consts.Tile) *MetaPlan {
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
	for i := range tiles {
		tiles[i] = plan.GetTile("wall")
	}
	if centerFloor {
		idx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: centerX, Y: centerY})
		tiles[idx] = plan.GetTile("floor")
	}
	return plan
}

func TestMetaPlan_randomPositionNear_中心が部屋内かつスポーン可能なら座標を返す(t *testing.T) {
	t.Parallel()

	centerX, centerY := consts.Tile(2), consts.Tile(2)
	plan := newRandomPositionNearTestPlan(true, centerX, centerY)
	room := gc.Rect{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 3, Y: 3}}

	// radius=0 なので候補座標は常に中心そのものになり、乱数に依存せず決定的に検証できる
	tx, ty, ok := plan.randomPositionNear(centerX, centerY, 0, room, w.World{}, 1)
	assert.True(t, ok)
	assert.Equal(t, centerX, tx)
	assert.Equal(t, centerY, ty)
}

func TestMetaPlan_randomPositionNear_スポーン不可なタイルなら試行回数内で失敗する(t *testing.T) {
	t.Parallel()

	centerX, centerY := consts.Tile(2), consts.Tile(2)
	plan := newRandomPositionNearTestPlan(false, centerX, centerY) // 中心を壁のままにする
	room := gc.Rect{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 3, Y: 3}}

	tx, ty, ok := plan.randomPositionNear(centerX, centerY, 0, room, w.World{}, 3)
	assert.False(t, ok)
	assert.Equal(t, consts.Tile(0), tx)
	assert.Equal(t, consts.Tile(0), ty)
}

func TestMetaPlan_randomPositionNear_部屋の範囲外の座標は候補から除外する(t *testing.T) {
	t.Parallel()

	// 中心(0,0)は床にしてスポーン可能にするが、部屋の範囲外なので選ばれてはいけない
	centerX, centerY := consts.Tile(0), consts.Tile(0)
	plan := newRandomPositionNearTestPlan(true, centerX, centerY)
	room := gc.Rect{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 3, Y: 3}}

	tx, ty, ok := plan.randomPositionNear(centerX, centerY, 0, room, w.World{}, 3)
	assert.False(t, ok)
	assert.Equal(t, consts.Tile(0), tx)
	assert.Equal(t, consts.Tile(0), ty)
}
