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

func TestBand_ShouldShiftEast(t *testing.T) {
	t.Parallel()

	b := worldstream.NewBand(100, 60, 3, 1) // 中央チャンクは帯ローカル [100,200)

	assert.False(t, b.ShouldShiftEast(199), "中央チャンク内では東シフトしない")
	assert.True(t, b.ShouldShiftEast(200), "中央チャンクを東へ出たら東シフト")
}

// TestBand_ShiftEast は東へ1回シフトする核心動作を固定する:
// 西端破棄・リベース・ExploredTiles追従・eastIndex前進・東端生成。
func TestBand_ShiftEast(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: 300, TileHeight: 60})) // K=3 * chunkW=100
	field := query.GetCurrentStageField(world)
	visState := query.GetVisionState(world)

	// プレイヤーは東チャンクへ踏み込んでいる（localX=210）
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 210, Y: 30}, "ash")
	require.NoError(t, err)
	// 西端チャンク [0,100) の敵 → 破棄される
	westEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 50, Y: 30}, "fireball")
	require.NoError(t, err)
	// 東チャンク [200,300) の敵 → 残ってリベースされる
	eastEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 250, Y: 30}, "fireball")
	require.NoError(t, err)

	// 探索済み: 中央(150,30)は生存→(50,30)へ、西(50,30)は破棄ゾーンへ落ちて消える
	field.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}: true,
		{Coord: consts.Coord[consts.Tile]{X: 50, Y: 30}}:  true,
	}
	// 視界も付け替え対象（チラつき防止のためクリアでなく平行移動する）
	visState.VisibleTiles = map[gc.GridElement]bool{{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}: true}

	b := worldstream.NewBand(100, 60, 3, 1)
	require.True(t, b.ShouldShiftEast(210), "前提: 東シフト条件を満たす")

	var gotCoord consts.Coord[consts.Chunk]
	var gotOffsetX, gotOffsetY consts.Tile
	gen := func(c consts.Coord[consts.Chunk], offsetX, offsetY consts.Tile) error {
		gotCoord = c
		gotOffsetX = offsetX
		gotOffsetY = offsetY
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

	// 生成呼び出し: X=eastIndex+K-1=3, offsetX=(K-1)*chunkW=200。1行帯なので Y=0, offsetY=0
	assert.Equal(t, consts.Coord[consts.Chunk]{X: 3, Y: 0}, gotCoord, "新チャンクの絶対座標")
	assert.Equal(t, consts.Tile(200), gotOffsetX, "東スラブのオフセット")
	assert.Equal(t, consts.Tile(0), gotOffsetY, "1行帯の縦オフセットは0")

	// ExploredTiles 追従: (150,30)→(50,30) 生存、(50,30)→(-50,30) は帯外で破棄
	assert.True(t, field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 50, Y: 30}}], "中央の探索済みは付け替わって残る")
	assert.False(t, field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}], "元キーは残らない")
	assert.Len(t, field.ExploredTiles, 1, "帯外に落ちた探索済みキーは捨てられる")

	// 視界も付け替えられる（クリアでなく平行移動。シフトフレームの暗転＝チラつきを防ぐ）
	assert.True(t, visState.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 50, Y: 30}}], "VisibleTiles も付け替わって残る")
	assert.False(t, visState.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 150, Y: 30}}], "元キーは残らない")

	// 壁配置が帯ローカル座標に対して変わったので、視界の強制再計算を要求する。
	// 立てないと VisionSystem のレイキャストキャッシュが旧壁配置の遮蔽結果を再利用し、幽霊影が出る
	assert.True(t, visState.ConsumePendingUpdate(), "シフト後は視界の強制再計算が要求される")
}
