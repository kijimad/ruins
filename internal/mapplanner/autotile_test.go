package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

func TestCalculateAutoTileIndex(t *testing.T) {
	t.Parallel()

	const dirtTileType = "Dirt"
	const floorTileType = "Floor"

	// テスト用の5x5マップを作成
	width, height := gc.Tile(5), gc.Tile(5)
	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]raw.TileRaw, int(width)*int(height)),
		RawMaster: createTestRawMaster(t),
	}

	// 全体を非土タイル（Floor）で初期化
	for i := range metaPlan.Tiles {
		tile, err := metaPlan.RawMaster.GetTile(floorTileType)
		require.NoError(t, err, "床タイル生成エラー")
		metaPlan.Tiles[i] = tile
	}

	// 中央（2,2）を土タイルに設定
	centerIdx := metaPlan.Level.XYTileIndex(gc.Tile(2), gc.Tile(2))
	dirtTile, err := metaPlan.RawMaster.GetTile(dirtTileType)
	require.NoError(t, err, "土タイル生成エラー")
	metaPlan.Tiles[centerIdx] = dirtTile

	// テストケース1: 孤立した土タイル
	autoTileIndex := metaPlan.CalculateAutoTileIndex(centerIdx, dirtTileType)
	if autoTileIndex != AutoTileIsolated {
		t.Errorf("孤立タイルの判定が間違っています。期待値: %s(%d), 実際: %s(%d)",
			AutoTileIsolated.String(), int(AutoTileIsolated),
			autoTileIndex.String(), int(autoTileIndex))
	}

	// テストケース2: 上に土タイルを追加（下だけが異なる状態）
	topIdx := metaPlan.Level.XYTileIndex(gc.Tile(2), gc.Tile(1))
	topDirt, err := metaPlan.RawMaster.GetTile(dirtTileType)
	require.NoError(t, err, "土タイル生成エラー")
	metaPlan.Tiles[topIdx] = topDirt

	autoTileIndex = metaPlan.CalculateAutoTileIndex(centerIdx, dirtTileType)

	// デバッグ情報を出力
	up := metaPlan.UpTile(centerIdx).Name == dirtTileType
	down := metaPlan.DownTile(centerIdx).Name == dirtTileType
	left := metaPlan.LeftTile(centerIdx).Name == dirtTileType
	right := metaPlan.RightTile(centerIdx).Name == dirtTileType
	t.Logf("デバッグ: 上=%t, 下=%t, 左=%t, 右=%t, ビットマスク=%d", up, down, left, right, int(autoTileIndex))

	if autoTileIndex != AutoTileUp {
		t.Errorf("上だけが同じタイルの判定が間違っています。期待値: %s(%d), 実際: %s(%d)",
			AutoTileUp.String(), int(AutoTileUp),
			autoTileIndex.String(), int(autoTileIndex))
	}

	// テストケース3: 右にも土タイルを追加（下左が異なる状態）
	rightIdx := metaPlan.Level.XYTileIndex(gc.Tile(3), gc.Tile(2))
	rightDirt, err := metaPlan.RawMaster.GetTile(dirtTileType)
	require.NoError(t, err, "土タイル生成エラー")
	metaPlan.Tiles[rightIdx] = rightDirt

	autoTileIndex = metaPlan.CalculateAutoTileIndex(centerIdx, dirtTileType)
	if autoTileIndex != AutoTileUpRight {
		t.Errorf("上右が同じタイルの判定が間違っています。期待値: %s(%d), 実際: %s(%d)",
			AutoTileUpRight.String(), int(AutoTileUpRight),
			autoTileIndex.String(), int(autoTileIndex))
	}

	// テストケース4: 全方向に土タイルを配置（中央タイル）
	bottomIdx := metaPlan.Level.XYTileIndex(gc.Tile(2), gc.Tile(3))
	leftIdx := metaPlan.Level.XYTileIndex(gc.Tile(1), gc.Tile(2))
	bottomDirt, err := metaPlan.RawMaster.GetTile(dirtTileType)
	require.NoError(t, err, "土タイル生成エラー")
	leftDirt, err := metaPlan.RawMaster.GetTile(dirtTileType)
	require.NoError(t, err, "土タイル生成エラー")
	metaPlan.Tiles[bottomIdx] = bottomDirt
	metaPlan.Tiles[leftIdx] = leftDirt

	autoTileIndex = metaPlan.CalculateAutoTileIndex(centerIdx, dirtTileType)
	if autoTileIndex != AutoTileCenter {
		t.Errorf("中央タイルの判定が間違っています。期待値: %s(%d), 実際: %s(%d)",
			AutoTileCenter.String(), int(AutoTileCenter),
			autoTileIndex.String(), int(autoTileIndex))
	}
}

func TestAutoTileIndexString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		index    AutoTileIndex
		expected string
	}{
		{AutoTileIsolated, "Isolated"},
		{AutoTileUp, "Up"},
		{AutoTileRight, "Right"},
		{AutoTileUpRight, "UpRight"},
		{AutoTileDown, "Down"},
		{AutoTileVertical, "Vertical"},
		{AutoTileDownRight, "DownRight"},
		{AutoTileUpDownRight, "UpDownRight"},
		{AutoTileLeft, "Left"},
		{AutoTileUpLeft, "UpLeft"},
		{AutoTileHorizontal, "Horizontal"},
		{AutoTileUpLeftRight, "UpLeftRight"},
		{AutoTileDownLeft, "DownLeft"},
		{AutoTileUpDownLeft, "UpDownLeft"},
		{AutoTileDownLeftRight, "DownLeftRight"},
		{AutoTileCenter, "Center"},
	}

	for _, tc := range testCases {
		actual := tc.index.String()
		if actual != tc.expected {
			t.Errorf("AutoTileIndex.String()が間違っています。インデックス: %d, 期待値: %s, 実際: %s",
				int(tc.index), tc.expected, actual)
		}
	}
}

func TestIsValidIndex(t *testing.T) {
	t.Parallel()

	// 3x3のテストマップ
	metaPlan := &MetaPlan{
		Tiles: make([]raw.TileRaw, 9), // 3x3 = 9タイル
	}

	testCases := []struct {
		idx      resources.TileIdx
		expected bool
	}{
		{resources.TileIdx(0), true},    // 有効な最小インデックス
		{resources.TileIdx(8), true},    // 有効な最大インデックス
		{resources.TileIdx(4), true},    // 有効な中央インデックス
		{resources.TileIdx(-1), false},  // 無効な負のインデックス
		{resources.TileIdx(9), false},   // 無効な範囲外インデックス
		{resources.TileIdx(100), false}, // 無効な大きすぎるインデックス
	}

	for _, tc := range testCases {
		actual := metaPlan.IsValidIndex(tc.idx)
		if actual != tc.expected {
			t.Errorf("IsValidIndex(%d)が間違っています。期待値: %t, 実際: %t",
				int(tc.idx), tc.expected, actual)
		}
	}
}

// createTestRawMaster はテスト用のrawマスターを作成する
func createTestRawMaster(t *testing.T) *raw.Master {
	t.Helper()

	rawData := `
[[Tiles]]
Name = "Floor"
Description = "床タイル"
Walkable = true

[[Tiles]]
Name = "Dirt"
Description = "土タイル"
Walkable = true

[[Tiles]]
Name = "Wall"
Description = "壁タイル"
Walkable = false
`

	master, err := raw.Load(rawData)
	require.NoError(t, err)

	return &master
}
