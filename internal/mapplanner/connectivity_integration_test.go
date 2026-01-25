package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
)

// TestPlannerChain_ValidateConnectivity はPlannerChainの接続性検証をテストする
func TestPlannerChain_ValidateConnectivity(t *testing.T) {
	t.Parallel()
	// 小さなテスト用マップを生成
	chain, err := NewSmallRoomPlanner(20, 20, 42) // 固定シードで再現可能
	require.NoError(t, err)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	err = chain.Plan()
	require.NoError(t, err)

	// プレイヤーのスタート位置を部屋の中心付近に設定
	var playerStartX, playerStartY int
	if len(chain.PlanData.Rooms) > 0 {
		room := chain.PlanData.Rooms[0]
		playerStartX = int(room.X1+room.X2) / 2
		playerStartY = int(room.Y1+room.Y2) / 2
	} else {
		// 部屋がない場合はマップの中央付近を使用
		playerStartX = 10
		playerStartY = 10

		// プレイヤーのスタート位置を確実に床にする
		idx := chain.PlanData.Level.XYTileIndex(gc.Tile(playerStartX), gc.Tile(playerStartY))
		chain.PlanData.Tiles[idx] = chain.PlanData.GetTile("floor")
	}

	// 接続性を検証（ワープポータルがない場合はErrNoWarpPortalが期待される）
	err = chain.ValidateConnectivity(playerStartX, playerStartY)
	if err != ErrNoWarpPortal {
		t.Errorf("Expected ErrNoWarpPortal for map without portals, got: %v", err)
	}
}

// TestCavePlanner_ValidateConnectivity は洞窟ビルダーの接続性をテストする
func TestCavePlanner_ValidateConnectivity(t *testing.T) {
	t.Parallel()
	// 洞窟マップを生成
	chain, err := NewCavePlanner(30, 30, 123)
	require.NoError(t, err)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	err = chain.Plan()
	require.NoError(t, err)

	// 床タイルを見つけてプレイヤーのスタート位置とする
	var playerStartX, playerStartY int
	var foundFloor bool

	width := int(chain.PlanData.Level.TileWidth)
	height := int(chain.PlanData.Level.TileHeight)

	// 中央付近から床タイルを探す
	for x := width/2 - 5; x < width/2+5 && !foundFloor; x++ {
		for y := height/2 - 5; y < height/2+5 && !foundFloor; y++ {
			if x >= 0 && x < width && y >= 0 && y < height {
				idx := chain.PlanData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
				if chain.PlanData.Tiles[idx].Walkable {
					playerStartX = x
					playerStartY = y
					foundFloor = true
				}
			}
		}
	}

	require.True(t, foundFloor, "Could not find a floor tile for player start position")

	// 接続性を検証（ワープポータルがない場合はErrNoWarpPortalが期待される）
	err = chain.ValidateConnectivity(playerStartX, playerStartY)
	if err != ErrNoWarpPortal {
		t.Errorf("Expected ErrNoWarpPortal for cave map without portals, got: %v", err)
	}
}

// TestPathFinder_WithPortals は手動でポータルを配置して接続性をテストする
func TestPathFinder_WithPortals(t *testing.T) {
	t.Parallel()
	// テスト用の小さなマップを作成
	chain := NewPlannerChain(10, 10, 1)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	chain.StartWith(&TestRoomPlanner{})
	err := chain.Plan()
	require.NoError(t, err)

	// プレイヤーのスタート位置（十字の中心）
	playerStartX, playerStartY := 5, 5

	// ワープポータルを配置（垂直通路上、到達可能な位置）
	warpIdx := chain.PlanData.Level.XYTileIndex(5, 2)
	chain.PlanData.Tiles[warpIdx] = chain.PlanData.GetTile("floor")
	// ワープポータルエンティティを追加
	chain.PlanData.WarpPortals = append(chain.PlanData.WarpPortals, WarpPortal{
		X:    5,
		Y:    2,
		Type: WarpPortalNext,
	})

	// 脱出ポータルを配置（到達不可能な位置：壁の中）
	escapeIdx := chain.PlanData.Level.XYTileIndex(1, 1)
	chain.PlanData.Tiles[escapeIdx] = chain.PlanData.GetTile("floor")
	// 脱出ポータルエンティティを追加
	chain.PlanData.WarpPortals = append(chain.PlanData.WarpPortals, WarpPortal{
		X:    1,
		Y:    1,
		Type: WarpPortalEscape,
	})

	// 接続性を検証（ワープポータルは到達可能だが脱出ポータルは到達不可能）
	// この場合、ワープポータルが到達可能なので接続性エラーは発生しない
	err = chain.ValidateConnectivity(playerStartX, playerStartY)
	if err != nil {
		t.Errorf("Expected no connectivity error when warp portal is reachable, got: %v", err)
	}

	t.Logf("Map connectivity validation passed with mixed portal reachability")
}

// TestRoomPlanner はテスト用の簡単な部屋ビルダー
type TestRoomPlanner struct{}

func (t *TestRoomPlanner) PlanInitial(planData *MetaPlan) error {
	// 中央に簡単な十字型の部屋を作成
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 垂直方向の通路
	for y := 1; y < height-1; y++ {
		idx := planData.Level.XYTileIndex(gc.Tile(width/2), gc.Tile(y))
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	// 水平方向の通路
	for x := 1; x < width-1; x++ {
		idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(height/2))
		planData.Tiles[idx] = planData.GetTile("floor")
	}

	return nil
}
