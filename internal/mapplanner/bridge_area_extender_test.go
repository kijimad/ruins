package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridgeAreaExtender_Extend(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	// 元のマップサイズ
	originalWidth := 50
	originalHeight := 30

	// MetaPlanを初期化
	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  gc.Tile(originalWidth),
			TileHeight: gc.Tile(originalHeight),
		},
		Tiles:     make([]raw.TileRaw, originalWidth*originalHeight),
		RawMaster: world.Resources.RawMaster.(*raw.Master),
		Rooms: []gc.Rect{
			{X1: 5, Y1: 5, X2: 10, Y2: 10}, // Y座標がシフトされるはず
		},
		NPCs: []NPCSpec{
			{X: 10, Y: 10}, // Y座標がシフトされるはず
		},
		Items: []ItemSpec{
			{X: 15, Y: 15}, // Y座標がシフトされるはず
		},
		Props: []PropsSpec{
			{X: 20, Y: 20, Name: "test_prop"}, // Y座標がシフトされるはず
		},
	}

	// 拡張実行
	err = extender.Extend(metaPlan)
	require.NoError(t, err)

	// マップサイズが拡張されたことを確認
	assert.Equal(t, gc.Tile(originalWidth), metaPlan.Level.TileWidth, "幅は変わらない")
	assert.Greater(t, int(metaPlan.Level.TileHeight), originalHeight, "高さが拡張される")

	// エンティティ座標がシフトされたことを確認（上部テンプレートの高さ分）
	topHeight := int(metaPlan.Level.TileHeight) - originalHeight - 28 // 下部テンプレートは28固定
	assert.Equal(t, gc.Tile(5+topHeight), metaPlan.Rooms[0].Y1, "Room Y1がシフトされる")
	assert.Equal(t, gc.Tile(10+topHeight), metaPlan.Rooms[0].Y2, "Room Y2がシフトされる")
	assert.Equal(t, gc.Tile(5), metaPlan.Rooms[0].X1, "Room X1は変わらない")
	assert.Equal(t, gc.Tile(10), metaPlan.Rooms[0].X2, "Room X2は変わらない")

	assert.Equal(t, 10+topHeight, metaPlan.NPCs[0].Y, "NPC Yがシフトされる")
	assert.Equal(t, 10, metaPlan.NPCs[0].X, "NPC Xは変わらない")

	assert.Equal(t, 15+topHeight, metaPlan.Items[0].Y, "Item Yがシフトされる")
	assert.Equal(t, 15, metaPlan.Items[0].X, "Item Xは変わらない")

	assert.Equal(t, 20+topHeight, metaPlan.Props[0].Y, "Prop Yがシフトされる")
	assert.Equal(t, 20, metaPlan.Props[0].X, "Prop Xは変わらない")

	// 橋情報（Exits/SpawnPoints/BridgeHints）が収集されたことを確認
	assert.NotEmpty(t, metaPlan.Exits, "Exitsが収集される")
	assert.NotEmpty(t, metaPlan.SpawnPoints, "SpawnPointsが収集される")
}

func TestBridgeAreaExtender_SmallMapWidth(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	// 幅20のマップ（街用）
	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  20,
			TileHeight: 20,
		},
		Tiles:     make([]raw.TileRaw, 20*20),
		RawMaster: world.Resources.RawMaster.(*raw.Master),
	}

	// 拡張実行
	err = extender.Extend(metaPlan)
	require.NoError(t, err)

	// 幅20用のテンプレートが使用されたことを確認
	assert.Equal(t, gc.Tile(20), metaPlan.Level.TileWidth, "幅20が維持される")
	assert.Greater(t, int(metaPlan.Level.TileHeight), 20, "高さが拡張される")
}

func TestBridgeAreaExtender_PreservesOriginalTiles(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	originalWidth := 50
	originalHeight := 10

	// 元のタイルに特定の値を設定
	originalTiles := make([]raw.TileRaw, originalWidth*originalHeight)
	for i := range originalTiles {
		originalTiles[i] = raw.TileRaw{Name: "test_tile"}
	}

	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  gc.Tile(originalWidth),
			TileHeight: gc.Tile(originalHeight),
		},
		Tiles:     originalTiles,
		RawMaster: world.Resources.RawMaster.(*raw.Master),
	}

	// 拡張実行
	err = extender.Extend(metaPlan)
	require.NoError(t, err)

	// 中央部分の元のタイルが保持されているか確認
	// 上部テンプレート高さを計算
	totalHeight := int(metaPlan.Level.TileHeight)
	bottomHeight := 28 // 下部テンプレート固定高さ
	topHeight := totalHeight - originalHeight - bottomHeight

	// 中央部分（元のマップ）の最初のタイルを確認
	centralStartIdx := topHeight * originalWidth
	assert.Equal(t, "test_tile", metaPlan.Tiles[centralStartIdx].Name, "元のタイルが保持される")
}

func TestBridgeAreaExtender_CollectsBridgeInfo(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  50,
			TileHeight: 20,
		},
		Tiles:     make([]raw.TileRaw, 50*20),
		RawMaster: world.Resources.RawMaster.(*raw.Master),
		Exits:     []maptemplate.ExitPlacement{}, // 初期は空
	}

	// 拡張実行
	err = extender.Extend(metaPlan)
	require.NoError(t, err)

	// 上部と下部のテンプレートから橋情報が収集される
	// 最低限の出口が追加されているはず
	assert.Greater(t, len(metaPlan.Exits), 0, "Exitsが収集される")

	// ExitIDが設定されているか確認
	for _, exit := range metaPlan.Exits {
		assert.NotEmpty(t, exit.ExitID, "ExitIDが設定される")
	}
}

func TestBridgeAreaExtender_NilMetaPlan(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	// nilのMetaPlanを渡すとパニックするはず
	assert.Panics(t, func() {
		_ = extender.Extend(nil)
	}, "nilのMetaPlanでパニックする")
}

func TestBridgeAreaExtender_EmptyTiles(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	// Tilesが空のMetaPlan（サイズとタイル配列が不一致）
	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  50,
			TileHeight: 30,
		},
		Tiles:     []raw.TileRaw{}, // 空（不正な状態）
		RawMaster: world.Resources.RawMaster.(*raw.Master),
	}

	// 拡張実行（空のタイル配列ではパニックする）
	assert.Panics(t, func() {
		_ = extender.Extend(metaPlan)
	}, "空のタイル配列でパニックする")
}

func TestBridgeAreaExtender_ZeroSize(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	// サイズ0のMetaPlan（不正な状態）
	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  0,
			TileHeight: 0,
		},
		Tiles:     []raw.TileRaw{},
		RawMaster: world.Resources.RawMaster.(*raw.Master),
	}

	// 拡張実行（幅0は対応していないのでエラーになる）
	err = extender.Extend(metaPlan)
	assert.Error(t, err, "対応していないマップ幅の場合はエラーになる")
	assert.Contains(t, err.Error(), "対応していないマップ幅です", "エラーメッセージに未対応幅が含まれる")
}

func TestBridgeAreaExtender_UnsupportedWidth(t *testing.T) {
	t.Parallel()

	extender, err := NewBridgeAreaExtender()
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	// 幅30のマップ（20でも50でもない）
	metaPlan := &MetaPlan{
		Level: resources.Level{
			TileWidth:  30,
			TileHeight: 20,
		},
		Tiles:     make([]raw.TileRaw, 30*20),
		RawMaster: world.Resources.RawMaster.(*raw.Master),
	}

	// 拡張実行（幅30は対応していないのでエラーになる）
	err = extender.Extend(metaPlan)
	assert.Error(t, err, "対応していないマップ幅の場合はエラーになる")
	assert.Contains(t, err.Error(), "対応していないマップ幅です", "エラーメッセージに未対応幅が含まれる")
}
