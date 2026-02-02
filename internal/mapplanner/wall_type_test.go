package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

func TestPlanData_GetWallType(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成（7x7）
	width, height := gc.Tile(7), gc.Tile(7)
	planData := &MetaPlan{
		Level: resources.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]raw.TileRaw, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]resources.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// テストケース1: WallTypeTop（下に床がある壁）
	// 座標系注意: XYTileIndex(tx Row, ty Col) → tx は X座標（横方向）、ty は Y座標（縦方向）
	// インデックス計算: ty * width + tx
	centerWallX, centerWallY := gc.Tile(3), gc.Tile(3)
	bottomFloorX, bottomFloorY := centerWallX, centerWallY+1 // 下の床（Y座標が大きくなる）

	centerWallIdx := planData.Level.XYTileIndex(centerWallX, centerWallY)
	bottomFloorIdx := planData.Level.XYTileIndex(bottomFloorX, bottomFloorY)

	planData.Tiles[bottomFloorIdx] = planData.GetTile("floor")

	// デバッグ情報を追加
	upFloor := planData.isFloorOrWarp(planData.UpTile(centerWallIdx))
	downFloor := planData.isFloorOrWarp(planData.DownTile(centerWallIdx))
	leftFloor := planData.isFloorOrWarp(planData.LeftTile(centerWallIdx))
	rightFloor := planData.isFloorOrWarp(planData.RightTile(centerWallIdx))

	wallType := planData.GetWallType(centerWallIdx)
	if wallType != WallTypeTop {
		t.Errorf("WallTypeTopの判定が間違っています。期待値: %s, 実際: %s\n上:%t, 下:%t, 左:%t, 右:%t",
			WallTypeTop.String(), wallType.String(), upFloor, downFloor, leftFloor, rightFloor)
	}

	// テストケース2: WallTypeRight（左に床がある壁）
	leftFloorX, leftFloorY := centerWallX-1, centerWallY // 左の床（X座標が小さくなる）
	leftFloorIdx := planData.Level.XYTileIndex(leftFloorX, leftFloorY)

	planData.Tiles[leftFloorIdx] = planData.GetTile("floor")
	planData.Tiles[bottomFloorIdx] = planData.GetTile("wall") // 前のテストケースをリセット

	wallType = planData.GetWallType(centerWallIdx)
	if wallType != WallTypeRight {
		t.Errorf("WallTypeRightの判定が間違っています。期待値: %s, 実際: %s", WallTypeRight.String(), wallType.String())
	}

	// テストケース3: WallTypeTopLeft（右下に床がある角壁）
	rightFloorX, rightFloorY := centerWallX+1, centerWallY // 右の床（X座標が大きくなる）
	downFloorX, downFloorY := centerWallX, centerWallY+1   // 下の床（Y座標が大きくなる）

	rightFloorIdx := planData.Level.XYTileIndex(rightFloorX, rightFloorY)
	downFloorIdx := planData.Level.XYTileIndex(downFloorX, downFloorY)

	planData.Tiles[rightFloorIdx] = planData.GetTile("floor")
	planData.Tiles[downFloorIdx] = planData.GetTile("floor")
	planData.Tiles[leftFloorIdx] = planData.GetTile("wall") // リセット

	wallType = planData.GetWallType(centerWallIdx)
	if wallType != WallTypeTopLeft {
		t.Errorf("WallTypeTopLeftの判定が間違っています。期待値: %s, 実際: %s", WallTypeTopLeft.String(), wallType.String())
	}

	// テストケース4: WallTypeGeneric（複雑なパターン）
	upFloorX, upFloorY := centerWallX, centerWallY-1 // 上の床（Y座標が小さくなる）
	upFloorIdx := planData.Level.XYTileIndex(upFloorX, upFloorY)
	planData.Tiles[upFloorIdx] = planData.GetTile("floor")

	wallType = planData.GetWallType(centerWallIdx) // 今は上、右、下に床がある状態
	if wallType != WallTypeGeneric {
		t.Errorf("WallTypeGenericの判定が間違っています。期待値: %s, 実際: %s", WallTypeGeneric.String(), wallType.String())
	}
}

func TestPlanData_GetWallType_WithWarpTiles(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成
	width, height := gc.Tile(5), gc.Tile(5)
	planData := &MetaPlan{
		Level: resources.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]raw.TileRaw, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]resources.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 壁と床を配置
	wallX, wallY := gc.Tile(2), gc.Tile(2)
	floorX, floorY := wallX, wallY+1 // 下に床を配置（Y座標が大きくなる）

	floorIdx := planData.Level.XYTileIndex(floorX, floorY)
	wallIdx := planData.Level.XYTileIndex(wallX, wallY)
	planData.Tiles[floorIdx] = planData.GetTile("floor")

	wallType := planData.GetWallType(wallIdx)
	if wallType != WallTypeTop {
		t.Errorf("床タイルに対するWallTypeTopの判定が間違っています。期待値: %s, 実際: %s", WallTypeTop.String(), wallType.String())
	}
}

func TestWallType_String(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		wallType WallType
		expected string
	}{
		{WallTypeTop, "Top"},
		{WallTypeBottom, "Bottom"},
		{WallTypeLeft, "Left"},
		{WallTypeRight, "Right"},
		{WallTypeTopLeft, "TopLeft"},
		{WallTypeTopRight, "TopRight"},
		{WallTypeBottomLeft, "BottomLeft"},
		{WallTypeBottomRight, "BottomRight"},
		{WallTypeGeneric, "Generic"},
	}

	for _, tc := range testCases {
		actual := tc.wallType.String()
		if actual != tc.expected {
			t.Errorf("WallType.String()の結果が間違っています。期待値: %s, 実際: %s", tc.expected, actual)
		}
	}
}

// TestCalculateWallAutoTileIndexMapping は壁用の13タイルマッピングをテストする
func TestCalculateWallAutoTileIndexMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*MetaPlan) resources.TileIdx
		expected AutoTileIndex
	}{
		{
			name: "全方向が壁の場合は9（右下コーナー、全方向と接続）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 全て壁（デフォルト）
				return mp.Level.XYTileIndex(3, 3)
			},
			expected: 9,
		},
		{
			name: "上だけ床の場合は1（上辺、下左右と接続）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				centerIdx := mp.Level.XYTileIndex(3, 3)
				x, y := mp.Level.XYTileCoord(centerIdx)
				upIdx := mp.Level.XYTileIndex(gc.Tile(x), gc.Tile(y-1))
				mp.Tiles[upIdx] = mp.GetTile("floor")
				return centerIdx
			},
			expected: 1,
		},
		{
			name: "下だけ床の場合は1（上辺、左右上と接続）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				centerIdx := mp.Level.XYTileIndex(3, 3)
				x, y := mp.Level.XYTileCoord(centerIdx)
				downIdx := mp.Level.XYTileIndex(gc.Tile(x), gc.Tile(y+1))
				mp.Tiles[downIdx] = mp.GetTile("floor")
				return centerIdx
			},
			expected: 1,
		},
		{
			name: "左と右と下が床の場合は6（横縞+左右枠）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				centerIdx := mp.Level.XYTileIndex(3, 3)
				x, y := mp.Level.XYTileCoord(centerIdx)
				downIdx := mp.Level.XYTileIndex(gc.Tile(x), gc.Tile(y+1))
				leftIdx := mp.Level.XYTileIndex(gc.Tile(x-1), gc.Tile(y))
				rightIdx := mp.Level.XYTileIndex(gc.Tile(x+1), gc.Tile(y))
				mp.Tiles[downIdx] = mp.GetTile("floor")
				mp.Tiles[leftIdx] = mp.GetTile("floor")
				mp.Tiles[rightIdx] = mp.GetTile("floor")
				return centerIdx
			},
			expected: 6,
		},
		{
			name: "左端の壁（左が範囲外=壁以外、右が床）は5（縦棒）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 左端の壁を選択
				leftEdgeIdx := mp.Level.XYTileIndex(0, 3)
				// 右に床を配置
				rightIdx := mp.Level.XYTileIndex(1, 3)
				mp.Tiles[rightIdx] = mp.GetTile("floor")
				return leftEdgeIdx
			},
			expected: 5, // 左が範囲外（壁以外）、右も床（ビットマスク10）→縦棒
		},
		{
			name: "右端の壁（右が範囲外=壁以外、左が床）は5（縦棒）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 右端の壁を選択
				rightEdgeIdx := mp.Level.XYTileIndex(6, 3)
				// 左に床を配置
				leftIdx := mp.Level.XYTileIndex(5, 3)
				mp.Tiles[leftIdx] = mp.GetTile("floor")
				return rightEdgeIdx
			},
			expected: 5, // 右が範囲外（壁以外）、左も床（ビットマスク10）→縦棒
		},
		{
			name: "上下が壁、右がvoid、左が床の場合は5（縦棒）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 右端の壁を選択（右側はvoid）
				centerIdx := mp.Level.XYTileIndex(6, 3)
				// 左に床を配置
				leftIdx := mp.Level.XYTileIndex(5, 3)
				mp.Tiles[leftIdx] = mp.GetTile("floor")
				// 上下は壁のまま（デフォルト）
				return centerIdx
			},
			expected: 5, // 上下が壁、右がvoid（壁以外）、左が床（ビットマスク10）→縦棒
		},
		{
			name: "voidタイルは壁として扱わない",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 中央に壁を配置
				centerIdx := mp.Level.XYTileIndex(3, 3)
				// 上下左右にvoidタイルを明示的に配置
				upIdx := mp.Level.XYTileIndex(3, 2)
				downIdx := mp.Level.XYTileIndex(3, 4)
				leftIdx := mp.Level.XYTileIndex(2, 3)
				rightIdx := mp.Level.XYTileIndex(4, 3)
				mp.Tiles[upIdx] = raw.TileRaw{Name: "void", BlockPass: false}
				mp.Tiles[downIdx] = raw.TileRaw{Name: "void", BlockPass: false}
				mp.Tiles[leftIdx] = raw.TileRaw{Name: "void", BlockPass: false}
				mp.Tiles[rightIdx] = raw.TileRaw{Name: "void", BlockPass: false}
				return centerIdx
			},
			expected: 6, // 全方向がvoid（壁ではない）→ ビットマスク15 → タイル6
		},
		{
			name: "範囲外は壁として扱わない（左端）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 左端の壁を選択（左は範囲外）
				leftEdgeIdx := mp.Level.XYTileIndex(0, 3)
				// 上下右は壁のまま（デフォルト）
				return leftEdgeIdx
			},
			expected: 5, // 左が範囲外（壁ではない）、右が壁 → ビットマスク8 → タイル5
		},
		{
			name: "範囲外は壁として扱わない（右端）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 右端の壁を選択（右は範囲外）
				rightEdgeIdx := mp.Level.XYTileIndex(6, 3)
				// 上下左は壁のまま（デフォルト）
				return rightEdgeIdx
			},
			expected: 5, // 右が範囲外（壁ではない）、左が壁 → ビットマスク2 → タイル5
		},
		{
			name: "範囲外は壁として扱わない（上端）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 上端の壁を選択（上は範囲外）
				topEdgeIdx := mp.Level.XYTileIndex(3, 0)
				// 下左右は壁のまま（デフォルト）
				return topEdgeIdx
			},
			expected: 1, // 上が範囲外（壁ではない）、下左右が壁 → ビットマスク1 → タイル1
		},
		{
			name: "範囲外は壁として扱わない（下端）",
			setup: func(mp *MetaPlan) resources.TileIdx {
				// 下端の壁を選択（下は範囲外）
				bottomEdgeIdx := mp.Level.XYTileIndex(3, 6)
				// 上左右は壁のまま（デフォルト）
				return bottomEdgeIdx
			},
			expected: 1, // 下が範囲外（壁ではない）、上左右が壁 → ビットマスク4 → タイル1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			width, height := gc.Tile(7), gc.Tile(7)
			mp := &MetaPlan{
				Level: resources.Level{
					TileWidth:  width,
					TileHeight: height,
				},
				Tiles:     make([]raw.TileRaw, int(width)*int(height)),
				RawMaster: CreateTestRawMaster(),
			}

			for i := range mp.Tiles {
				mp.Tiles[i] = mp.GetTile("wall")
			}

			testIdx := tt.setup(mp)

			actual := mp.CalculateWallAutoTileIndex(testIdx)

			if actual != tt.expected {
				t.Errorf("期待値: %d, 実際: %d", tt.expected, actual)
			}
		})
	}
}
