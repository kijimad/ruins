package worldstream

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// Band はアクティブ帯を管理する。アクティブ帯は横 cols 列・縦 rows 行の隣接チャンクを
// 1連続座標空間に並べた単一マップ。南北は北へ無限にストリーミングし、東西は cols に有界で
// ストリーミングしない。
//
// プレイヤーは常に中央チャンク行に保たれ、中央行を北へ出るとシフトする。シフトは北端行の
// 生成・南端行の破棄・リベースからなる。これにより帯ローカル座標は常に 0..rows*chunkH に
// 収まり、既存の単一マップ機構を変えずに無限北進を実現する。北は画面の上すなわち -Y。
type Band struct {
	chunkW     consts.Tile  // 1チャンクの幅。横スロットへの配置に使う。構築後不変
	chunkH     consts.Tile  // 1チャンクの高さ。構築後不変
	cols       consts.Chunk // X方向のチャンク列数。有界。構築後不変
	rows       consts.Chunk // Y方向のチャンク行数。奇数で中央チャンクを持つ。構築後不変
	northIndex consts.Chunk // 北進したチャンク数。0以上で北へ進むほど増える。シフトで変化
}

// ChunkGen はチャンク座標 c の地形を帯ローカルの (offsetX, offsetY) 位置へ生成・配置する。
// 呼び出し側が引数からの決定的生成と mapspawner.SpawnAt を実装する。
// worldstream を mapplanner/mapspawner に依存させないための注入点。
type ChunkGen func(c consts.Coord[consts.Chunk], offsetX, offsetY consts.Tile) error

// NewBand は幅 chunkW・高さ chunkH のチャンクを横 cols 列・縦 rows 行並べた帯を northIndex=0 で作る。
// rows は奇数を推奨する。cols=1 なら1列の帯になる。
func NewBand(chunkW, chunkH consts.Tile, cols, rows consts.Chunk) *Band {
	return NewBandAt(chunkW, chunkH, cols, rows, 0)
}

// NewBandAt は northIndex を指定して帯を作る。セーブからの復元で使う。
func NewBandAt(chunkW, chunkH consts.Tile, cols, rows, northIndex consts.Chunk) *Band {
	return &Band{chunkW: chunkW, chunkH: chunkH, cols: cols, rows: rows, northIndex: northIndex}
}

// ChunkW は1チャンクの幅を返す。
func (b *Band) ChunkW() consts.Tile { return b.chunkW }

// Cols は帯の横のチャンク列数を返す。
func (b *Band) Cols() consts.Chunk { return b.cols }

// NorthIndex は北進したチャンク数を返す。北へ進むほど増える。
func (b *Band) NorthIndex() consts.Chunk { return b.northIndex }

// BandOriginY は帯ローカル Y=0 すなわち北端が指す絶対 Y。
func (b *Band) BandOriginY() consts.AbsTileY { return consts.BandOriginY(b.northIndex, b.chunkH) }

// Width は帯の総幅。帯ローカル X の有効範囲は [0, Width())。
func (b *Band) Width() consts.Tile { return b.cols.Tiles(b.chunkW) }

// Rows は Y 方向のチャンク行数を返す。
func (b *Band) Rows() consts.Chunk { return b.rows }

// Height は帯の総高。帯ローカル Y の有効範囲は [0, Height())。
func (b *Band) Height() consts.Tile { return b.rows.Tiles(b.chunkH) }

// centerSlot は中央チャンクの行スロット番号。rows が奇数なら真ん中。
func (b *Band) centerSlot() consts.Chunk { return b.rows / 2 }

// ShouldShiftNorth はプレイヤーの帯ローカル Y が中央チャンクを北へ出たかを返す。判定はヒステリシスを持つ。
// 北は -Y なので、中央行より上、すなわち localY が中央行の上端未満へ入ったらシフトする。
func (b *Band) ShouldShiftNorth(playerLocalY consts.Tile) bool {
	return playerLocalY < b.centerSlot().Tiles(b.chunkH)
}

// ShiftNorth は帯を北へ1チャンク進める。
// 南端行の破棄 → リベース → 座標キー Map 追従 → northIndex 前進 → 北端行の生成。
func (b *Band) ShiftNorth(world w.World, gen ChunkGen) error {
	// 1. 南端の行を全列破棄する。シフトは中央より北でしか発火しないのでプレイヤーは破棄範囲へ入らないが、
	// 中央行が南端に重なる退化帯に備え KeepPlayer で保険をかける
	RemoveEntitiesInYRange(world, (b.rows - 1).Tiles(b.chunkH), b.rows.Tiles(b.chunkH), KeepPlayer(world))
	// 2. リベース。全エンティティを南へ chunkH ずらしてプレイヤーを中央へ戻す
	TranslateAllEntities(world, 0, b.chunkH)
	// 3. 座標キー Map を追従させる。
	b.rebaseCoordMaps(world, b.chunkH)
	// 4. northIndex 前進 → 新しい北端の行を全列生成・配置する。北端の絶対チャンク Y は負へ伸びる
	b.northIndex++
	newChunkY := -b.northIndex
	for cx := range b.cols {
		if err := gen(consts.Coord[consts.Chunk]{X: cx, Y: newChunkY}, cx.Tiles(b.chunkW), 0); err != nil {
			return err
		}
	}
	return nil
}

// rebaseCoordMaps はリベースに伴い座標キーの Map を追従させる。
// 永続の ExploredTiles はキーを平行移動し、揮発キャッシュはクリアして次フレーム再構築させる。
func (b *Band) rebaseCoordMaps(world w.World, dy consts.Tile) {
	field := query.GetCurrentStageField(world)
	if field == nil {
		return
	}
	inBand := func(g gc.GridElement) bool {
		return g.Y >= 0 && g.Y < b.Height()
	}
	// リベースは純粋な座標シフトなので、座標キーの Map はすべてキー付け替えで追従させる。
	// 視界の VisibleTiles と LightSourceCache もクリアでなく付け替える。こうするとシフトと同じ
	// フレームの描画で有効なまま保て、チャンク境界越え時のチラつきを防ぐ。チラつきは1フレームの
	// 暗転として現れる。次フレームの VisionSystem がどのみち再計算するが、その1フレームの穴を無くす。
	field.ExploredTiles = translateTileKeyMap(field.ExploredTiles, 0, dy, inBand)
	vs := query.GetVisionState(world)
	vs.VisibleTiles = translateTileKeyMap(vs.VisibleTiles, 0, dy, inBand)
	vs.LightSourceCache = translateTileKeyMap(vs.LightSourceCache, 0, dy, inBand)
	// シフトで帯ローカル座標に対する壁配置が変わるため、視界の強制再計算を要求する。
	// これにより VisionSystem は壁配置依存のレイキャストキャッシュも破棄する
	vs.RequestUpdate()
	query.InvalidateSpatialIndex(world)
}

// translateTileKeyMap は GridElement キーの map を dx,dy 平行移動した新しい map を返す。
// keep が false を返すキーは捨てる。帯外に落ちたキーが該当する。keep が nil のときはフィルタせず全キーを通す。
func translateTileKeyMap[V any](src map[gc.GridElement]V, dx, dy consts.Tile, keep func(gc.GridElement) bool) map[gc.GridElement]V {
	if src == nil {
		return nil
	}
	dst := make(map[gc.GridElement]V, len(src))
	for k, v := range src {
		nk := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: k.X + dx, Y: k.Y + dy}}
		if keep != nil && !keep(nk) {
			continue
		}
		dst[nk] = v
	}
	return dst
}
