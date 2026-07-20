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

// RecalcSeamAutotile はチャンク境界 x=boundaryX をまたぐ2列のタイルのオートタイルを、
// 両チャンクの実タイルのエンティティを見て再計算する。
//
// チャンクは独立生成されるため、生成時は境界列の隣を void 扱いして端スプライトになり、
// 継ぎ目が見える。接合後に境界列を再計算して継ぎ目を消す。boundaryX-1 が西チャンク東端、
// boundaryX が東チャンク西端。
// 描画は SpriteKey で sprite をフェッチするため、SpriteKey の差し替えだけで見た目が直る。
//
// 境界の両側にタイルが揃っている「内部境界」だけを処理する。片側が空なら何もしない。帯の最西端・
// 最東端で隣チャンクが無い場合が該当する。これにより呼び出し側は東西どちらの境界かを気にせず
// 両境界を無条件に呼べる。東シフトは西境界、西シフトは東境界が実境界になる。
func RecalcSeamAutotile(world w.World, boundaryX consts.Tile) {
	// 境界周辺の列 boundaryX-2..boundaryX+1 のタイルを位置引きできるよう集める。
	// 再計算対象の左右隣 boundaryX-2 と boundaryX+1 まで含める
	tiles := make(map[gc.GridElement]ecs.Entity)
	hasWest, hasEast := false, false
	// 帯の継ぎ目再計算は現ステージ(帯)のタイルだけを対象にする
	q := query.ActiveFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world).Query()
	for q.Next() {
		e := q.Entity()
		g := *world.Components.GridElement.Get(e)
		if g.X >= boundaryX-2 && g.X <= boundaryX+1 {
			tiles[g] = e
			if g.X == boundaryX-1 {
				hasWest = true
			}
			if g.X == boundaryX {
				hasEast = true
			}
		}
	}
	// 片側が空なら帯端の外周であり、直すべき継ぎ目は無い
	if !hasWest || !hasEast {
		return
	}

	nameOf := func(g gc.GridElement) (string, bool) {
		e, ok := tiles[g]
		if !ok || !world.Components.Name.Has(e) {
			return "", false
		}
		return world.Components.Name.Get(e).Name, true
	}

	// 境界の2列を再計算する
	for _, e := range tiles {
		g := *world.Components.GridElement.Get(e)
		if g.X != boundaryX-1 && g.X != boundaryX {
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
