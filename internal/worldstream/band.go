package worldstream

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// Band はアクティブ帯（K個の隣接チャンクを1連続座標空間に並べた「単一マップ」）を管理する。
//
// プレイヤーは常に中央チャンクに保たれ、中央チャンクを東へ出るとシフトする（東端生成・西端破棄・
// リベース）。これにより帯ローカル座標は常に 0..K*chunkW に収まり、既存の単一マップ機構を
// 変えずに無限東進を実現する。詳細設計は docs/design/20260717_60.md §2。
type Band struct {
	chunkW    consts.Tile // 1チャンクの幅（タイル）。構築後不変
	k         int         // 帯のチャンク数（奇数。中央チャンクを持つ）。構築後不変
	eastIndex int         // 東進したチャンク数（帯西端チャンクの絶対インデックス）。シフトで変化
}

// ChunkGen は絶対チャンクインデックスの地形を帯ローカルの offsetX 位置へ生成・配置する。
// 呼び出し側が (runSeed, chunkIndex) からの決定的生成と mapspawner.SpawnAt を実装する。
// worldstream を mapplanner/mapspawner に依存させないための注入点。
type ChunkGen func(chunkIndex int, offsetX consts.Tile) error

// NewBand は幅 chunkW・チャンク数 k（奇数推奨）の帯を eastIndex=0 で作る。
func NewBand(chunkW consts.Tile, k int) *Band {
	return NewBandAt(chunkW, k, 0)
}

// NewBandAt は eastIndex を指定して帯を作る。セーブからの復元（Phase 6）で使う。
func NewBandAt(chunkW consts.Tile, k, eastIndex int) *Band {
	return &Band{chunkW: chunkW, k: k, eastIndex: eastIndex}
}

// ChunkW は1チャンクの幅（タイル）を返す。
func (b *Band) ChunkW() consts.Tile { return b.chunkW }

// K は帯のチャンク数を返す。
func (b *Band) K() int { return b.k }

// EastIndex は東進したチャンク数（帯西端チャンクの絶対インデックス）を返す。
func (b *Band) EastIndex() int { return b.eastIndex }

// BandOriginX は帯ローカル X=0 が指す絶対 X。
func (b *Band) BandOriginX() AbsTileX { return BandOriginX(b.eastIndex, b.chunkW) }

// Width は帯の総幅（タイル）。帯ローカル X の有効範囲は [0, Width())。
func (b *Band) Width() consts.Tile { return b.chunkW * consts.Tile(b.k) }

// centerSlot は中央チャンクのスロット番号（K が奇数なら真ん中）。
func (b *Band) centerSlot() int { return b.k / 2 }

// ShouldShiftEast はプレイヤーの帯ローカル X が中央チャンクを東へ出たかを返す（ヒステリシス）。
func (b *Band) ShouldShiftEast(playerLocalX consts.Tile) bool {
	return playerLocalX >= consts.Tile(b.centerSlot()+1)*b.chunkW
}

// ShouldShiftWest はプレイヤーが中央チャンクを西へ出たかを返す（短い寄り道の復帰時のみ）。
func (b *Band) ShouldShiftWest(playerLocalX consts.Tile) bool {
	return playerLocalX < consts.Tile(b.centerSlot())*b.chunkW
}

// ShiftEast は帯を東へ1チャンク進める（§2.2 shiftEast の合成）。
// 西端チャンク破棄 → リベース → 座標キー Map 追従 → eastIndex 前進 → 東端チャンク生成。
func (b *Band) ShiftEast(world w.World, gen ChunkGen) error {
	// 1. 西端チャンク破棄（前線が呑む）。プレイヤー・隊員は残す
	RemoveEntitiesInXRange(world, 0, b.chunkW, KeepPlayerAndSquad(world))
	// 2. リベース：全エンティティを西へ chunkW（プレイヤーを中央へ戻す）
	TranslateAllEntities(world, -b.chunkW, 0)
	// 3. 座標キー Map を追従させる（§2.4）
	b.rebaseCoordMaps(world, -b.chunkW)
	// 4. eastIndex 前進 → 新しい東端チャンクを生成・配置
	b.eastIndex++
	newChunkIndex := b.eastIndex + b.k - 1
	offsetX := consts.Tile(b.k-1) * b.chunkW
	return gen(newChunkIndex, offsetX)
}

// ShiftWest は帯を西へ1チャンク戻す（ShiftEast の対称。短い寄り道からの復帰時のみ）。
func (b *Band) ShiftWest(world w.World, gen ChunkGen) error {
	// 東端チャンク破棄
	RemoveEntitiesInXRange(world, (consts.Tile(b.k)-1)*b.chunkW, b.Width(), KeepPlayerAndSquad(world))
	// リベース：全エンティティを東へ chunkW
	TranslateAllEntities(world, b.chunkW, 0)
	b.rebaseCoordMaps(world, b.chunkW)
	// eastIndex 後退 → 西端チャンクを生成・配置
	b.eastIndex--
	newChunkIndex := b.eastIndex
	return gen(newChunkIndex, 0)
}

// rebaseCoordMaps はリベースに伴い座標キーの Map を追従させる（§2.4）。
// 永続の ExploredTiles はキーを平行移動し、揮発キャッシュはクリアして次フレーム再構築させる。
func (b *Band) rebaseCoordMaps(world w.World, dx consts.Tile) {
	d := query.GetDungeon(world)
	if d == nil {
		return
	}
	inBand := func(g gc.GridElement) bool {
		return g.X >= 0 && g.X < b.Width()
	}
	// 永続記憶：キー付け替え（帯外に落ちたキーは捨てる）
	d.ExploredTiles = translateTileKeyMap(d.ExploredTiles, dx, 0, inBand)
	// 揮発キャッシュ：毎移動/毎フレーム再計算されるためクリアで足りる
	d.VisibleTiles = make(map[gc.GridElement]bool)
	d.LightSourceCache = make(map[gc.GridElement]gc.LightInfo)
	query.InvalidateSpatialIndex(world)
}

// translateTileKeyMap は GridElement キーの map を (dx,dy) 平行移動した新しい map を返す。
// keep が false を返すキー（帯外に落ちたもの）は捨てる。
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
