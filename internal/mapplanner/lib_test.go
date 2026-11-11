package mapplanner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
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
	tiles := make([]rawschema.Tile, int(width)*int(height))
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

// newEdgeTileTestPlan は境界判定検証用の3x3マップを作る。全タイルを floor にする
func newEdgeTileTestPlan() *MetaPlan {
	width, height := consts.Tile(3), consts.Tile(3)
	plan := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]rawschema.Tile, int(width)*int(height)),
		RawMaster: CreateTestRawMaster(),
	}
	for i := range plan.Tiles {
		plan.Tiles[i] = plan.GetTile(consts.TileNameFloor)
	}
	return plan
}

func TestMetaPlan_UpTile_上端は境界外のvoidタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	topIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 0})

	tile := plan.UpTile(topIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameVoid), tile)
}

func TestMetaPlan_UpTile_境界外でなければ上のタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	centerIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})

	tile := plan.UpTile(centerIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameFloor), tile)
}

func TestMetaPlan_DownTile_下端は境界外のvoidタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	bottomIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 2})

	tile := plan.DownTile(bottomIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameVoid), tile)
}

func TestMetaPlan_DownTile_境界外でなければ下のタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	centerIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})

	tile := plan.DownTile(centerIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameFloor), tile)
}

func TestMetaPlan_LeftTile_左端は境界外のvoidタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	leftIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 0, Y: 1})

	tile := plan.LeftTile(leftIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameVoid), tile)
}

func TestMetaPlan_LeftTile_境界外でなければ左のタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	centerIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})

	tile := plan.LeftTile(centerIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameFloor), tile)
}

func TestMetaPlan_RightTile_右端は境界外のvoidタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	rightIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 2, Y: 1})

	tile := plan.RightTile(rightIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameVoid), tile)
}

func TestMetaPlan_RightTile_境界外でなければ右のタイルを返す(t *testing.T) {
	t.Parallel()

	plan := newEdgeTileTestPlan()
	centerIdx := plan.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})

	tile := plan.RightTile(centerIdx)

	assert.Equal(t, plan.GetTile(consts.TileNameFloor), tile)
}

func TestMetaPlan_GetTile_RawMaster未設定ならpanicする(t *testing.T) {
	t.Parallel()

	plan := &MetaPlan{}

	assert.PanicsWithValue(t, "RawMaster is not set; tile generation from TOML is required", func() {
		plan.GetTile(consts.TileNameFloor)
	})
}

func TestMetaPlan_GetTile_存在しないタイル名ならpanicする(t *testing.T) {
	t.Parallel()

	plan := &MetaPlan{RawMaster: CreateTestRawMaster()}

	assert.PanicsWithValue(t, "tile generation error: key not found: no_such_tile in Tiles", func() {
		plan.GetTile("no_such_tile")
	})
}
