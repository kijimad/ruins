package overworld

import (
	"strconv"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// RecalcChunkSeams はチャンクを接合した後に、そのチャンクの東西南北4つの境界のオートタイルを
// 再計算して継ぎ目を消す。チャンクは独立生成され、生成時は境界の隣を void 扱いして端スプライトに
// なるため継ぎ目が見える。接合後に境界の2ラインを実タイルのエンティティから再計算して繋ぐ。
// 描画は SpriteKey で sprite をフェッチするので、SpriteKey の差し替えだけで見た目が直る。
//
// 東西境界は x=offsetX と x=offsetX+chunkW、南北境界は y=offsetY と y=offsetY+chunkH。
// boundaryX-1 が西チャンク東端、boundaryX が東チャンク西端で、南北も同様の対応になる。
//
// 各境界は両側にタイルが揃う内部境界だけを処理し、片側が空の帯端では何もしない。帯の最端や
// 行が1つの帯の上下がこれに当たる。よって呼び出し側は4境界を無条件にまとめて呼べる。東シフトは
// 西境界、西シフトは東境界が実境界になり、行が増えれば南北境界も同じ扱いになる。
func RecalcChunkSeams(world w.World, offsetX, offsetY, chunkW, chunkH consts.Tile) {
	byX := func(g gc.GridElement) consts.Tile { return g.X }
	byY := func(g gc.GridElement) consts.Tile { return g.Y }
	recalcSeamAutotileAlong(world, offsetX, byX)
	recalcSeamAutotileAlong(world, offsetX+chunkW, byX)
	recalcSeamAutotileAlong(world, offsetY, byY)
	recalcSeamAutotileAlong(world, offsetY+chunkH, byY)
}

// RecalcAutotileInXRange は X 範囲 [loX, hiX) のタイルのオートタイルを、実エンティティの
// 近傍から再計算する。地物の層がタイルを置換した後に呼び、置換タイル自身と周囲の土の
// 添字を実状態へ揃える。近傍参照のため範囲の外側1列も集めるが、再計算は範囲内に限る。
func RecalcAutotileInXRange(world w.World, loX, hiX consts.Tile) {
	tiles := make(map[gc.GridElement]ecs.Entity)
	q := query.ActiveFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world).Query()
	for q.Next() {
		e := q.Entity()
		g := *world.Components.GridElement.Get(e)
		if g.X >= loX-1 && g.X <= hiX {
			tiles[g] = e
		}
	}
	nameOf := func(g gc.GridElement) (string, bool) {
		e, ok := tiles[g]
		if !ok || !world.Components.Name.Has(e) {
			return "", false
		}
		return world.Components.Name.Get(e).Name, true
	}
	for _, e := range tiles {
		g := *world.Components.GridElement.Get(e)
		if g.X < loX || g.X >= hiX {
			continue
		}
		recalcTileAutotile(world, e, g, nameOf)
	}
}

// recalcSeamAutotileAlong は境界をまたぐ2ラインのオートタイル再計算の共通実装。
// axis が返す座標軸に沿って boundary-1 と boundary の2ラインを対象にする。
// x 境界(東西チャンク間)なら axis は X を、y 境界(南北チャンク間)なら Y を返す。
func recalcSeamAutotileAlong(world w.World, boundary consts.Tile, axis func(g gc.GridElement) consts.Tile) {
	// 境界周辺の boundary-2..boundary+1 のタイルを位置引きできるよう集める。
	// 再計算対象の両隣 boundary-2 と boundary+1 まで含める
	tiles := make(map[gc.GridElement]ecs.Entity)
	hasLow, hasHigh := false, false
	// 帯の継ぎ目再計算は現ステージ(帯)のタイルだけを対象にする
	q := query.ActiveFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world).Query()
	for q.Next() {
		e := q.Entity()
		g := *world.Components.GridElement.Get(e)
		c := axis(g)
		if c >= boundary-2 && c <= boundary+1 {
			tiles[g] = e
			if c == boundary-1 {
				hasLow = true
			}
			if c == boundary {
				hasHigh = true
			}
		}
	}
	// 片側が空なら帯端の外周であり、直すべき継ぎ目は無い
	if !hasLow || !hasHigh {
		return
	}

	nameOf := func(g gc.GridElement) (string, bool) {
		e, ok := tiles[g]
		if !ok || !world.Components.Name.Has(e) {
			return "", false
		}
		return world.Components.Name.Get(e).Name, true
	}

	// 境界の2ラインを再計算する
	for _, e := range tiles {
		g := *world.Components.GridElement.Get(e)
		c := axis(g)
		if c != boundary-1 && c != boundary {
			continue
		}
		recalcTileAutotile(world, e, g, nameOf)
	}
}

// recalcTileAutotile は1タイルのオートタイル SpriteKey を4近傍から再計算する。
func recalcTileAutotile(world w.World, e ecs.Entity, g gc.GridElement, nameOf func(gc.GridElement) (string, bool)) {
	if !world.Components.Name.Has(e) {
		return
	}
	self := world.Components.Name.Get(e).Name
	sr := world.Components.SpriteRender.Get(e)
	base, ok := autotileBase(sr.SpriteKey)
	if !ok {
		return // オートタイルでないタイルはスキップする。数値サフィックスが無い void 等が該当する
	}

	// CalculateAutoTileIndex と同じビット割り当て: 上1・右2・下4・左8
	bit := 0
	if n, ok := nameOf(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: g.X, Y: g.Y - 1}}); ok && n == self {
		bit |= 1
	}
	if n, ok := nameOf(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: g.X + 1, Y: g.Y}}); ok && n == self {
		bit |= 2
	}
	if n, ok := nameOf(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: g.X, Y: g.Y + 1}}); ok && n == self {
		bit |= 4
	}
	if n, ok := nameOf(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: g.X - 1, Y: g.Y}}); ok && n == self {
		bit |= 8
	}
	sr.SpriteKey = base + "_" + strconv.Itoa(bit)
}

// autotileBase は "dirt_15" → ("dirt", true) を返す。サフィックスが数値でなければ false。
func autotileBase(key string) (string, bool) {
	i := strings.LastIndex(key, "_")
	if i < 0 {
		return "", false
	}
	if _, err := strconv.Atoi(key[i+1:]); err != nil {
		return "", false
	}
	return key[:i], true
}
