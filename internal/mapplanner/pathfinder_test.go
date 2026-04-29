package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestPlanData はテスト用のPlanDataを作成する
func createTestPlanData(_, _ int) *MetaPlan {
	width := 5  // 固定値を使用
	height := 5 // 固定値を使用
	tileCount := width * height
	tiles := make([]raw.TileRaw, tileCount)

	// 一時的なMetaPlanインスタンスを作成
	tempPlan := &MetaPlan{
		Level: resources.Level{
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
	if pf.IsWalkable(-1, 0) {
		t.Error("Expected (-1, 0) to be not walkable")
	}
	if pf.IsWalkable(0, -1) {
		t.Error("Expected (0, -1) to be not walkable")
	}
	if pf.IsWalkable(5, 0) {
		t.Error("Expected (5, 0) to be not walkable")
	}
	if pf.IsWalkable(0, 5) {
		t.Error("Expected (0, 5) to be not walkable")
	}

	// 壁タイルテスト（デフォルト）
	if pf.IsWalkable(1, 1) {
		t.Error("Expected wall tile to be not walkable")
	}

	// 床タイルに変更してテスト
	idx := planData.Level.XYTileIndex(1, 1)
	planData.Tiles[idx] = planData.GetTile("floor")
	if !pf.IsWalkable(1, 1) {
		t.Error("Expected floor tile to be walkable")
	}

	// ワープタイルテスト
	idx = planData.Level.XYTileIndex(2, 2)
	planData.Tiles[idx] = planData.GetTile("floor")
	if !pf.IsWalkable(2, 2) {
		t.Error("Expected warp next tile to be walkable")
	}

	// 脱出タイルテスト
	idx = planData.Level.XYTileIndex(3, 3)
	planData.Tiles[idx] = planData.GetTile("floor")
	if !pf.IsWalkable(3, 3) {
		t.Error("Expected warp escape tile to be walkable")
	}
}

func TestPathFinder_FindPath_SimplePath(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// 簡単な一直線のパスを作成
	// (1,1) -> (1,2) -> (1,3)
	for y := 1; y <= 3; y++ {
		idx := planData.Level.XYTileIndex(1, consts.Tile(y))
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	path := pf.FindPath(1, 1, 1, 3)

	expectedLength := 3 // スタート、中間、ゴール
	if len(path) != expectedLength {
		t.Errorf("Expected path length %d, got %d", expectedLength, len(path))
	}

	// パスの内容を検証
	expected := []consts.Coord[int]{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 1, Y: 3}}
	for i, pos := range expected {
		if i >= len(path) || path[i].X != pos.X || path[i].Y != pos.Y {
			t.Errorf("Expected position %d to be (%d, %d), got (%d, %d)",
				i, pos.X, pos.Y, path[i].X, path[i].Y)
		}
	}
}

func TestPathFinder_FindPath_NoPath(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// スタート地点のみ床にする（ゴールは壁のまま）
	idx := planData.Level.XYTileIndex(1, 1)
	planData.Tiles[idx] = planData.GetTile("floor")

	path := pf.FindPath(1, 1, 3, 3)

	if len(path) != 0 {
		t.Errorf("Expected no path, got path of length %d", len(path))
	}
}

func TestPathFinder_FindPath_LShapedPath(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// L字型のパスを作成
	// (1,1) -> (1,2) -> (2,2) -> (3,2)
	positions := []consts.Coord[int]{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2}}
	for _, pos := range positions {
		idx := planData.Level.XYTileIndex(consts.Tile(pos.X), consts.Tile(pos.Y))
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	path := pf.FindPath(1, 1, 3, 2)

	if len(path) != 4 {
		t.Errorf("Expected path length 4, got %d", len(path))
	}

	// スタートとゴールが正しいことを確認
	if path[0].X != 1 || path[0].Y != 1 {
		t.Errorf("Expected start at (1,1), got (%d,%d)", path[0].X, path[0].Y)
	}
	if path[len(path)-1].X != 3 || path[len(path)-1].Y != 2 {
		t.Errorf("Expected goal at (3,2), got (%d,%d)",
			path[len(path)-1].X, path[len(path)-1].Y)
	}
}

func TestPathFinder_IsReachable(t *testing.T) {
	t.Parallel()
	planData := createTestPlanData(5, 5)
	pf := NewPathFinder(planData)

	// パスを作成
	positions := []consts.Coord[int]{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 2, Y: 2}}
	for _, pos := range positions {
		idx := planData.Level.XYTileIndex(consts.Tile(pos.X), consts.Tile(pos.Y))
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	// 到達可能なテスト
	if !pf.IsReachable(1, 1, 2, 2) {
		t.Error("Expected (1,1) to (2,2) to be reachable")
	}

	// 到達不可能なテスト
	if pf.IsReachable(1, 1, 3, 3) {
		t.Error("Expected (1,1) to (3,3) to be not reachable")
	}
}

// TestFindPlayerStartPosition_AvoidsNPCs はプレイヤーのスポーン位置がNPCと重複しないことを検証する
func TestFindPlayerStartPosition_AvoidsNPCs(t *testing.T) {
	t.Parallel()

	// 十分な広さのマップを作成
	width, height := 15, 15
	tileCount := width * height
	tiles := make([]raw.TileRaw, tileCount)
	planData := &MetaPlan{
		Level: resources.Level{
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
			idx := planData.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))
			tiles[idx] = planData.GetTile("floor")
		}
	}

	// ポータルを配置
	planData.NextPortals = []consts.Coord[int]{{X: 1, Y: 1}}

	// 中央付近にNPCを配置してプレイヤーの最優先候補位置を塞ぐ
	planData.NPCs = []NPCSpec{
		{Coord: consts.Coord[int]{X: width / 2, Y: height / 2}, Name: "test"},
	}

	pf := NewPathFinder(planData)
	pos, err := pf.FindPlayerStartPosition()
	require.NoError(t, err)

	// プレイヤーのスポーン位置がNPCと重複しないことを検証
	for _, npc := range planData.NPCs {
		assert.False(t, pos.X == npc.X && pos.Y == npc.Y,
			"プレイヤーのスポーン位置(%d,%d)がNPC位置(%d,%d)と重複しています", pos.X, pos.Y, npc.X, npc.Y)
	}
}
