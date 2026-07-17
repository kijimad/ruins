package overworld

import (
	"strconv"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// RecalcSeamAutotile はチャンク境界 x=boundaryX をまたぐ2列のタイルのオートタイルを、
// 両チャンクの実タイル（エンティティ）を見て再計算する。
//
// チャンクは独立生成されるため、生成時は境界列の隣を void 扱いして端スプライトになり、
// 継ぎ目が見える。接合後に境界列（boundaryX-1 = 西チャンク東端、boundaryX = 東チャンク西端）を
// 再計算して継ぎ目を消す（設計 docs/design/20260717_60.md §5）。
// 描画は SpriteKey で sprite をフェッチするため、SpriteKey の差し替えだけで見た目が直る。
//
// 境界の両側にタイルが揃っている「内部境界」だけを処理する。片側が空（帯の最西端・最東端で
// 隣チャンクが無い）なら何もしない。これにより呼び出し側は東西どちらの境界かを気にせず
// 両境界を無条件に呼べる（東シフトは西境界、西シフトは東境界が実境界になる）。
func RecalcSeamAutotile(world w.World, boundaryX consts.Tile) {
	// 境界周辺（列 boundaryX-2..boundaryX+1）のタイルを位置引きできるよう集める。
	// 再計算対象の左右隣（boundaryX-2 / boundaryX+1）まで含める
	tiles := make(map[gc.GridElement]ecs.Entity)
	hasWest, hasEast := false, false
	query := ecs.NewFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world.ECS).Query()
	for query.Next() {
		e := query.Entity()
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
		return // オートタイルでないタイル（数値サフィックス無し。void 等）はスキップ
	}

	// CalculateAutoTileIndex と同じビット割り当て: 上1・右2・下4・左8
	bit := 0
	if n, ok := nameOf(gc.GridElement{X: g.X, Y: g.Y - 1}); ok && n == self {
		bit |= 1
	}
	if n, ok := nameOf(gc.GridElement{X: g.X + 1, Y: g.Y}); ok && n == self {
		bit |= 2
	}
	if n, ok := nameOf(gc.GridElement{X: g.X, Y: g.Y + 1}); ok && n == self {
		bit |= 4
	}
	if n, ok := nameOf(gc.GridElement{X: g.X - 1, Y: g.Y}); ok && n == self {
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
