package worldstream

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// Band はアクティブ帯を管理する。アクティブ帯は K個の隣接チャンクを1連続座標空間に並べた単一マップ。
//
// プレイヤーは常に中央チャンクに保たれ、中央チャンクを東へ出るとシフトする。シフトは東端生成・
// 西端破棄・リベースからなる。これにより帯ローカル座標は常に 0..K*chunkW に収まり、既存の単一
// マップ機構を変えずに無限東進を実現する。詳細設計は docs/design/20260717_60.md §2。
type Band struct {
	chunkW    consts.Tile // 1チャンクの幅。構築後不変
	k         ChunkX      // 帯のチャンク数。奇数で中央チャンクを持つ。構築後不変
	eastIndex ChunkX      // 東進したチャンク数。帯西端チャンクの絶対インデックス。シフトで変化
}

// ChunkGen は絶対チャンクインデックスの地形を帯ローカルの offsetX 位置へ生成・配置する。
// 呼び出し側が (runSeed, chunkIndex) からの決定的生成と mapspawner.SpawnAt を実装する。
// worldstream を mapplanner/mapspawner に依存させないための注入点。
type ChunkGen func(chunkIndex ChunkX, offsetX consts.Tile) error

// NewBand は幅 chunkW、チャンク数 k の帯を eastIndex=0 で作る。k は奇数を推奨する。
func NewBand(chunkW consts.Tile, k ChunkX) *Band {
	return NewBandAt(chunkW, k, 0)
}

// NewBandAt は eastIndex を指定して帯を作る。セーブからの復元で使う。
func NewBandAt(chunkW consts.Tile, k, eastIndex ChunkX) *Band {
	return &Band{chunkW: chunkW, k: k, eastIndex: eastIndex}
}

// ChunkW は1チャンクの幅を返す。
func (b *Band) ChunkW() consts.Tile { return b.chunkW }

// K は帯のチャンク数を返す。
func (b *Band) K() ChunkX { return b.k }

// EastIndex は東進したチャンク数を返す。帯西端チャンクの絶対インデックス。
func (b *Band) EastIndex() ChunkX { return b.eastIndex }

// BandOriginX は帯ローカル X=0 が指す絶対 X。
func (b *Band) BandOriginX() AbsTileX { return BandOriginX(b.eastIndex, b.chunkW) }

// Width は帯の総幅。帯ローカル X の有効範囲は [0, Width())。
func (b *Band) Width() consts.Tile { return b.k.Tiles(b.chunkW) }

// centerSlot は中央チャンクのスロット番号。K が奇数なら真ん中。
func (b *Band) centerSlot() ChunkX { return b.k / 2 }

// ShouldShiftEast はプレイヤーの帯ローカル X が中央チャンクを東へ出たかを返す。判定はヒステリシスを持つ。
func (b *Band) ShouldShiftEast(playerLocalX consts.Tile) bool {
	return playerLocalX >= (b.centerSlot() + 1).Tiles(b.chunkW)
}

// ShouldShiftWest はプレイヤーが中央チャンクを西へ出たかを返す。短い寄り道からの復帰時のみ使う。
func (b *Band) ShouldShiftWest(playerLocalX consts.Tile) bool {
	return playerLocalX < b.centerSlot().Tiles(b.chunkW)
}

// ShiftEast は帯を東へ1チャンク進める。§2.2 shiftEast の合成。
// 西端チャンク破棄 → リベース → 座標キー Map 追従 → eastIndex 前進 → 東端チャンク生成。
func (b *Band) ShiftEast(world w.World, gen ChunkGen) error {
	// 1. 西端チャンクを破棄する。前線が呑む。プレイヤーと隊員は残す
	RemoveEntitiesInXRange(world, 0, b.chunkW, KeepPlayerAndSquad(world))
	// 2. リベース。全エンティティを西へ chunkW ずらしてプレイヤーを中央へ戻す
	TranslateAllEntities(world, -b.chunkW, 0)
	// 3. 座標キー Map を追従させる。§2.4
	b.rebaseCoordMaps(world, -b.chunkW)
	// 4. eastIndex 前進 → 新しい東端チャンクを生成・配置
	b.eastIndex++
	newChunkIndex := b.eastIndex + b.k - 1
	offsetX := (b.k - 1).Tiles(b.chunkW)
	return gen(newChunkIndex, offsetX)
}

// ShiftWest は帯を西へ1チャンク戻す。ShiftEast の対称で、短い寄り道からの復帰時のみ使う。
func (b *Band) ShiftWest(world w.World, gen ChunkGen) error {
	// 東端チャンク破棄
	RemoveEntitiesInXRange(world, (b.k - 1).Tiles(b.chunkW), b.Width(), KeepPlayerAndSquad(world))
	// リベース：全エンティティを東へ chunkW
	TranslateAllEntities(world, b.chunkW, 0)
	b.rebaseCoordMaps(world, b.chunkW)
	// eastIndex 後退 → 西端チャンクを生成・配置
	b.eastIndex--
	newChunkIndex := b.eastIndex
	return gen(newChunkIndex, 0)
}

// rebaseCoordMaps はリベースに伴い座標キーの Map を追従させる。§2.4。
// 永続の ExploredTiles はキーを平行移動し、揮発キャッシュはクリアして次フレーム再構築させる。
func (b *Band) rebaseCoordMaps(world w.World, dx consts.Tile) {
	d := query.GetDungeon(world)
	if d == nil {
		return
	}
	inBand := func(g gc.GridElement) bool {
		return g.X >= 0 && g.X < b.Width()
	}
	// リベースは純粋な座標シフトなので、座標キーの Map はすべてキー付け替えで追従させる。
	// 視界の VisibleTiles と LightSourceCache もクリアでなく付け替える。こうするとシフトと同じ
	// フレームの描画で有効なまま保て、チャンク境界越え時のチラつきを防ぐ。チラつきは1フレームの
	// 暗転として現れる。次フレームの VisionSystem がどのみち再計算するが、その1フレームの穴を無くす。
	d.ExploredTiles = translateTileKeyMap(d.ExploredTiles, dx, 0, inBand)
	d.VisibleTiles = translateTileKeyMap(d.VisibleTiles, dx, 0, inBand)
	d.LightSourceCache = translateTileKeyMap(d.LightSourceCache, dx, 0, inBand)
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
		nk := gc.GridElement{X: k.X + dx, Y: k.Y + dy}
		if keep != nil && !keep(nk) {
			continue
		}
		dst[nk] = v
	}
	return dst
}
