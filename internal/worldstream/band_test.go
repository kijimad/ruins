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

func TestBand_ShouldShiftNorth(t *testing.T) {
	t.Parallel()

	b := worldstream.NewBand(60, 100, 1, 3) // 中央チャンク行は帯ローカル [100,200)

	assert.False(t, b.ShouldShiftNorth(100), "中央チャンク内では北シフトしない")
	assert.True(t, b.ShouldShiftNorth(99), "中央チャンクを北へ出たら北シフト")
}

// TestBand_ShiftNorth は北へ1回シフトする核心動作を固定する:
// 南端破棄・リベース・ExploredTiles追従・northIndex前進・北端生成。
func TestBand_ShiftNorth(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: 60, TileHeight: 300})) // K=3 * chunkH=100
	field := query.GetCurrentStageField(world)
	visState := query.GetVisionState(world)

	// プレイヤーは北チャンクへ踏み込んでいる（localY=90）
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 30, Y: 90}, "ash")
	require.NoError(t, err)
	// 南端チャンク [200,300) の敵 → 破棄される
	southEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 30, Y: 250}, "fireball")
	require.NoError(t, err)
	// 北チャンク [0,100) の敵 → 残ってリベースされる
	northEnemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 30, Y: 50}, "fireball")
	require.NoError(t, err)

	// 探索済み: 中央(30,150)は生存→(30,250)へ、南寄り(30,250)は破棄ゾーンへ落ちて消える
	field.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 30, Y: 150}}: true,
		{Coord: consts.Coord[consts.Tile]{X: 30, Y: 250}}: true,
	}
	// 視界も付け替え対象（チラつき防止のためクリアでなく平行移動する）
	visState.VisibleTiles = map[gc.GridElement]bool{{Coord: consts.Coord[consts.Tile]{X: 30, Y: 150}}: true}
	// 光源キャッシュもチラつき防止のため付け替え対象。値は問わずキーの追従だけ見る
	visState.LightSourceCache = map[gc.GridElement]gc.LightInfo{{Coord: consts.Coord[consts.Tile]{X: 30, Y: 150}}: {Darkness: 0.5}}

	b := worldstream.NewBand(60, 100, 1, 3)
	require.True(t, b.ShouldShiftNorth(90), "前提: 北シフト条件を満たす")

	var gotCoord consts.Coord[consts.Chunk]
	var gotOffsetX, gotOffsetY consts.Tile
	gen := func(c consts.Coord[consts.Chunk], offsetX, offsetY consts.Tile) error {
		gotCoord = c
		gotOffsetX = offsetX
		gotOffsetY = offsetY
		// 北端に新チャンクのタイルを1枚だけ置く（マーカー）
		world.Components.GridElement.NewEntity(&gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: offsetY + 5}})
		return nil
	}

	require.NoError(t, b.ShiftNorth(world, gen))

	// northIndex 前進・原点更新。北は -Y なので原点は負へ伸びる
	assert.Equal(t, 1, int(b.NorthIndex()), "northIndex が1つ進む")
	assert.Equal(t, consts.AbsTileY(-100), b.BandOriginY(), "帯原点が chunkH ぶん北へ")

	// 南端チャンクの敵は破棄
	assert.False(t, world.ECS.Alive(southEnemy), "南端チャンクの敵は破棄される")

	// リベース：プレイヤー 90→190（中央へ）、北敵 50→150
	assert.Equal(t, consts.Tile(190), world.Components.GridElement.Get(player).Y, "プレイヤーは中央へ引き戻される")
	assert.Equal(t, consts.Tile(150), world.Components.GridElement.Get(northEnemy).Y, "北敵もリベースされる")

	// 生成呼び出し: 北端の絶対チャンク Y=-northIndex=-1, offsetY=0。1列帯なので X=0, offsetX=0
	assert.Equal(t, consts.Coord[consts.Chunk]{X: 0, Y: -1}, gotCoord, "新チャンクの絶対座標")
	assert.Equal(t, consts.Tile(0), gotOffsetX, "1列帯の横オフセットは0")
	assert.Equal(t, consts.Tile(0), gotOffsetY, "北スラブのオフセットは0")

	// ExploredTiles 追従: (30,150)→(30,250) 生存、(30,250)→(30,350) は帯外で破棄
	assert.True(t, field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 250}}], "中央の探索済みは付け替わって残る")
	assert.False(t, field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 150}}], "元キーは残らない")
	assert.NotContains(t, field.ExploredTiles, gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 350}}, "帯外へリベースされた探索済みはキーごと破棄される")
	assert.Len(t, field.ExploredTiles, 1, "帯外に落ちた探索済みキーは捨てられる")

	// 視界も付け替えられる（クリアでなく平行移動。シフトフレームの暗転＝チラつきを防ぐ）
	assert.True(t, visState.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 250}}], "VisibleTiles も付け替わって残る")
	assert.False(t, visState.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 150}}], "元キーは残らない")

	// LightSourceCache も同じく付け替えられる。VisibleTiles と揃ってチラつきを防ぐ要のロジック
	_, lightMoved := visState.LightSourceCache[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 250}}]
	assert.True(t, lightMoved, "LightSourceCache も付け替わって残る")
	assert.NotContains(t, visState.LightSourceCache, gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 30, Y: 150}}, "元キーは残らない")

	// 壁配置が帯ローカル座標に対して変わったので、視界の強制再計算を要求する。
	// 立てないと VisionSystem のレイキャストキャッシュが旧壁配置の遮蔽結果を再利用し、幽霊影が出る
	assert.True(t, visState.ConsumePendingUpdate(), "シフト後は視界の強制再計算が要求される")
}
