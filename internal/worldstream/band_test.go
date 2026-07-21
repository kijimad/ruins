package worldstream_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBand_ShouldShift_ヒステリシス(t *testing.T) {
	t.Parallel()

	b := worldstream.NewBand(100, 3) // 中央チャンクは帯ローカル [100,200)

	assert.False(t, b.ShouldShiftEast(199), "中央チャンク内では東シフトしない")
	assert.True(t, b.ShouldShiftEast(200), "中央チャンクを東へ出たら東シフト")
	assert.False(t, b.ShouldShiftWest(100), "中央チャンク西端では西シフトしない")
	assert.True(t, b.ShouldShiftWest(99), "中央チャンクを西へ出たら西シフト")
}

// TestBand_ShiftEast は東へ1回シフトする核心動作を固定する:
// 西端破棄・リベース・ExploredTiles追従・eastIndex前進・東端生成。
func TestBand_ShiftEast(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	visState := query.GetVisionState(world)
	d.Level = gc.Level{TileWidth: 300, TileHeight: 60} // K=3 * chunkW=100

	// プレイヤーは東チャンクへ踏み込んでいる（localX=210）
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 210, Y: 30}, "Ash")
	require.NoError(t, err)
	// 西端チャンク [0,100) の敵 → 破棄される
	westEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 50, Y: 30}, "火の玉")
	require.NoError(t, err)
	// 東チャンク [200,300) の敵 → 残ってリベースされる
	eastEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 250, Y: 30}, "火の玉")
	require.NoError(t, err)

	// 探索済み: 中央(150,30)は生存→(50,30)へ、西(50,30)は破棄ゾーンへ落ちて消える
	d.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}: true,
		{Coord: consts.Coord[consts.Tile]{X: 50, Y: 30}}:  true,
	}
	// 視界も付け替え対象（チラつき防止のためクリアでなく平行移動する）
	visState.VisibleTiles = map[gc.GridElement]bool{{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}: true}

	b := worldstream.NewBand(100, 3)
	require.True(t, b.ShouldShiftEast(210), "前提: 東シフト条件を満たす")

	var gotChunkIndex consts.Chunk
	var gotOffsetX consts.Tile
	gen := func(chunkIndex consts.Chunk, offsetX consts.Tile) error {
		gotChunkIndex = chunkIndex
		gotOffsetX = offsetX
		// 東端に新チャンクのタイルを1枚だけ置く（マーカー）
		world.Components.GridElement.NewEntity(&gc.GridElement{Coord: consts.Coord[consts.Tile]{X: offsetX + 5, Y: 10}})
		return nil
	}

	require.NoError(t, b.ShiftEast(world, gen))

	// eastIndex 前進・原点更新
	assert.Equal(t, 1, int(b.EastIndex()), "eastIndex が1つ進む")
	assert.Equal(t, consts.AbsTileX(100), b.BandOriginX(), "帯原点が chunkW ぶん東へ")

	// 西端チャンクの敵は破棄
	assert.False(t, world.ECS.Alive(westEnemy), "西端チャンクの敵は破棄される")

	// リベース：プレイヤー 210→110（中央へ）、東敵 250→150
	assert.Equal(t, consts.Tile(110), world.Components.GridElement.Get(player).X, "プレイヤーは中央へ引き戻される")
	assert.Equal(t, consts.Tile(150), world.Components.GridElement.Get(eastEnemy).X, "東敵もリベースされる")

	// 生成呼び出し: chunkIndex=eastIndex+K-1=3, offsetX=(K-1)*chunkW=200
	assert.Equal(t, 3, int(gotChunkIndex), "新チャンクの絶対インデックス")
	assert.Equal(t, consts.Tile(200), gotOffsetX, "東スラブのオフセット")

	// ExploredTiles 追従: (150,30)→(50,30) 生存、(50,30)→(-50,30) は帯外で破棄
	assert.True(t, d.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 50, Y: 30}}], "中央の探索済みは付け替わって残る")
	assert.False(t, d.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}], "元キーは残らない")
	assert.Len(t, d.ExploredTiles, 1, "帯外に落ちた探索済みキーは捨てられる")

	// 視界も付け替えられる（クリアでなく平行移動。シフトフレームの暗転＝チラつきを防ぐ）
	assert.True(t, visState.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 50, Y: 30}}], "VisibleTiles も付け替わって残る")
	assert.False(t, visState.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}], "元キーは残らない")
}

// TestBand_ShiftWest は西へ1回シフトする対称動作を固定する（短い寄り道の復帰）。
func TestBand_ShiftWest(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.Level = gc.Level{TileWidth: 300, TileHeight: 60}

	// プレイヤーは西チャンクへ踏み込んでいる（localX=90）
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 90, Y: 30}, "Ash")
	require.NoError(t, err)
	// 東端チャンク [200,300) の敵 → 破棄される
	eastEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 250, Y: 30}, "火の玉")
	require.NoError(t, err)
	// 西チャンク [0,100) の敵 → 残ってリベースされる
	westEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 50, Y: 30}, "火の玉")
	require.NoError(t, err)

	b := worldstream.NewBandAt(100, 3, 1) // 一度東へ進んだ状態から西へ戻る
	require.True(t, b.ShouldShiftWest(90), "前提: 西シフト条件を満たす")

	var gotChunkIndex consts.Chunk
	var gotOffsetX consts.Tile
	gen := func(chunkIndex consts.Chunk, offsetX consts.Tile) error {
		gotChunkIndex = chunkIndex
		gotOffsetX = offsetX
		return nil
	}

	require.NoError(t, b.ShiftWest(world, gen))

	assert.Equal(t, 0, int(b.EastIndex()), "eastIndex が1つ戻る")
	assert.False(t, world.ECS.Alive(eastEnemy), "東端チャンクの敵は破棄される")
	assert.Equal(t, consts.Tile(190), world.Components.GridElement.Get(player).X, "プレイヤーは東へリベースされ中央へ")
	assert.Equal(t, consts.Tile(150), world.Components.GridElement.Get(westEnemy).X, "西敵もリベースされる")
	assert.Equal(t, 0, int(gotChunkIndex), "新しい西端チャンクの絶対インデックス")
	assert.Equal(t, consts.Tile(0), gotOffsetX, "西スラブのオフセットは0")
}

// TestBand_ShiftWest_eastIndex0はエラー は、ラン開始地点で西シフトを呼ぶと eastIndex を
// 負にせずエラーを返すことを固定する。maybeShift のガードとは別に Band 自身が誤用を弾く。
func TestBand_ShiftWest_eastIndex0はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	b := worldstream.NewBand(100, 3) // eastIndex=0
	genCalled := false
	gen := func(_ consts.Chunk, _ consts.Tile) error {
		genCalled = true
		return nil
	}

	err := b.ShiftWest(world, gen)
	require.Error(t, err, "eastIndex=0 での西シフトはエラー")
	assert.Equal(t, 0, int(b.EastIndex()), "eastIndex は負にならない")
	assert.False(t, genCalled, "生成は呼ばれない")
}
