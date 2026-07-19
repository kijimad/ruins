package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestPlanData はテスト用のPlanDataを作成する
func createTestPlanData(_, _ int) *MetaPlan {
	width := 5  // 固定値を使用
	height := 5 // 固定値を使用
	tileCount := width * height
	tiles := make([]oapi.Tile, tileCount)

	// 一時的なMetaPlanインスタンスを作成
	tempPlan := &MetaPlan{
		Level: gc.Level{
			TileWidth:  consts.Tile(width),
			TileHeight: consts.Tile(height),
		},
		Tiles:     tiles,
		RawMaster: CreateTestRawMaster(),
	}

	// デフォルトで全て壁にする
	for i := range tiles {
		tiles[i] = tempPlan.GetTile("wall")
	}

	return tempPlan
}

func TestPathFinder_IsWalkable(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// 境界外テスト
	assert.False(t, pf.IsWalkable(-1, 0), "Expected (-1, 0) to be not walkable")
	assert.False(t, pf.IsWalkable(0, -1), "Expected (0, -1) to be not walkable")
	assert.False(t, pf.IsWalkable(5, 0), "Expected (5, 0) to be not walkable")
	assert.False(t, pf.IsWalkable(0, 5), "Expected (0, 5) to be not walkable")

	// 壁タイルテスト（デフォルト）
	assert.False(t, pf.IsWalkable(1, 1), "Expected wall tile to be not walkable")

	// 床タイルに変更してテスト
	idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})
	planData.Tiles[idx] = planData.GetTile("floor")
	assert.True(t, pf.IsWalkable(1, 1), "Expected floor tile to be walkable")

	// ワープタイルテスト
	idx = planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 2, Y: 2})
	planData.Tiles[idx] = planData.GetTile("floor")
	assert.True(t, pf.IsWalkable(2, 2), "Expected warp next tile to be walkable")

	// 脱出タイルテスト
	idx = planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 3})
	planData.Tiles[idx] = planData.GetTile("floor")
	assert.True(t, pf.IsWalkable(3, 3), "Expected warp escape tile to be walkable")
}

func TestPathFinder_FindPath_SimplePath(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// 簡単な一直線のパスを作成
	// (1,1) -> (1,2) -> (1,3)
	for y := 1; y <= 3; y++ {
		idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: consts.Tile(y)})
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	path := pf.FindPath(1, 1, 1, 3)

	expectedLength := 3 // スタート、中間、ゴール
	require.Len(t, path, expectedLength, "パス長が期待値と異なる")

	// パスの内容を検証
	expected := []consts.Coord[int]{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 1, Y: 3}}
	for i, pos := range expected {
		assert.Equal(t, pos.X, path[i].X, "位置%dのXが期待値と異なる", i)
		assert.Equal(t, pos.Y, path[i].Y, "位置%dのYが期待値と異なる", i)
	}
}

func TestPathFinder_FindPath_NoPath(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// スタート地点のみ床にする（ゴールは壁のまま）
	idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})
	planData.Tiles[idx] = planData.GetTile("floor")

	path := pf.FindPath(1, 1, 3, 3)

	assert.Empty(t, path, "パスが存在しないはずなのに見つかった")
}

func TestPathFinder_FindPath_LShapedPath(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// L字型のパスを作成
	// (1,1) -> (1,2) -> (2,2) -> (3,2)
	positions := []consts.Coord[int]{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}}
	for _, pos := range positions {
		idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(pos.X), Y: consts.Tile(pos.Y)})
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	path := pf.FindPath(1, 1, 3, 2)

	require.Len(t, path, 4, "パス長が期待値と異なる")

	// スタートとゴールが正しいことを確認
	assert.Equal(t, 1, path[0].X, "スタートのXが期待値と異なる")
	assert.Equal(t, 1, path[0].Y, "スタートのYが期待値と異なる")
	assert.Equal(t, 3, path[len(path)-1].X, "ゴールのXが期待値と異なる")
	assert.Equal(t, 2, path[len(path)-1].Y, "ゴールのYが期待値と異なる")
}

func TestPathFinder_IsReachable(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// パスを作成
	positions := []consts.Coord[int]{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 2, Y: 2}}
	for _, pos := range positions {
		idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(pos.X), Y: consts.Tile(pos.Y)})
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	// 到達可能なテスト
	assert.True(t, pf.IsReachable(1, 1, 2, 2), "Expected (1,1) to (2,2) to be reachable")

	// 到達不可能なテスト
	assert.False(t, pf.IsReachable(1, 1, 3, 3), "Expected (1,1) to (3,3) to be not reachable")
}

// TestFindPlayerStartPosition_AvoidsNPCs はプレイヤーのスポーン位置がNPCと重複しないことを検証する
func TestFindPlayerStartPosition_AvoidsNPCs(t *testing.T) {
	t.Parallel()

	// 十分な広さのマップを作成
	width, height := 15, 15
	tileCount := width * height
	tiles := make([]oapi.Tile, tileCount)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  consts.Tile(width),
			TileHeight: consts.Tile(height),
		},
		Tiles:     tiles,
		RawMaster: CreateTestRawMaster(),
	}
	for i := range tiles {
		tiles[i] = planData.GetTile("wall")
	}
	// 内側を床にする
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
			tiles[idx] = planData.GetTile("floor")
		}
	}

	// ポータルを配置
	planData.NextPortals = []consts.Coord[consts.Tile]{{X: 1, Y: 1}}

	// 中央付近にNPCを配置してプレイヤーの最優先候補位置を塞ぐ
	planData.NPCs = []NPCSpec{
		{Coord: consts.Coord[consts.Tile]{X: consts.Tile(width / 2), Y: consts.Tile(height / 2)}, Name: "test"},
	}

	pf := NewPathFinder(planData)
	pos, err := pf.FindPlayerStartPosition()
	require.NoError(t, err)

	// プレイヤーのスポーン位置がNPCと重複しないことを検証
	for _, npc := range planData.NPCs {
		assert.False(t, pos.X == int(npc.X) && pos.Y == int(npc.Y),
			"プレイヤーのスポーン位置(%d,%d)がNPC位置(%d,%d)と重複しています", pos.X, pos.Y, npc.X, npc.Y)
	}
}
