package overworld

import (
	"fmt"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 屋外の散布は wasteland チャンクの開けた地表へ prop を密に撒く feature。疎な Placement と違い
// doc 70 の密度場を屋外へ流用し、density 確率で個数を決める。他フィーチャの後に評価し、道が
// タイルを置換した後の実状態を読んで dirt かつ BlockPass 非占有のタイルだけに置く。

// scatterEntry は散布物の1候補。実スプライトのある prop を優先する。Big は通行を塞ぐ大物の印で、
// 位相格子と経路マスクの対象になる。Ref が空文字なら「置かない」を表し、null 重みとして扱う。
// Satellites があれば anchor 相対の小クラスタを1回で置く。
type scatterEntry struct {
	Ref        string
	Weight     int
	Big        bool
	Satellites []relSpot
}

// scatterCatalog は zone ごとの散布定義。Density は面積あたり期待個数の係数で、
// count = round(area * Density)。個数を反復乱数でなく密度確率で決めるので純関数のまま。
type scatterCatalog struct {
	Zone    outdoorZone
	Density float64
	Entries []scatterEntry
}

// outdoorZone は wasteland チャンクの人の手の残り具合。近傍の道・集落・市街地までの距離で決まる。
// 実体は文字列にし、%v やログで数値でなく種別名が出て読みやすい。iota の連番は割に合わないので使わない。
type outdoorZone string

const (
	// zoneRoadside は道・集落・市街に近い。放置車両・ドラム缶・瓦礫・自販機など人工物が主。
	zoneRoadside outdoorZone = "roadside"
	// zoneWild は奥地。枯れ木・木立・岩・石柱など自然物が主。
	zoneWild outdoorZone = "wild"
)

// 位相格子と経路マスクの定数。どちらも Big の大物だけに効き、通行を塞がない小物は制約しない。
const (
	// scatterBigPhaseMod は大物を乗せる位相の法。候補タイルを (x + scatterBigPhaseK*y) mod P へ分け、
	// Big は位相 0 のタイルにだけ乗せる。実効密度が 1/P に希釈され、P が通行性のダイヤルになる。
	scatterBigPhaseMod consts.Tile = 3
	scatterBigPhaseK   consts.Tile = 2
	// scatterRouteBuffer は経路マスクのバッファ幅。隊商の横断レーンを確保するため、算出した経路の
	// 直線からこのタイル数以内では Big の大物を置かない。
	scatterRouteBuffer consts.Tile = 2
	// scatterRoadsideRange は道沿いゾーンの半径。集落・市街の当選チャンクからこのチャンク数以内を
	// 道沿いにする。道は集落間を結ぶので、集落の近傍が道沿いの帯を兼ねる。
	scatterRoadsideRange consts.Chunk = 1
)

// roadsideCatalog は道沿いゾーンの散布定義。人工物を主に密度高め。Ref 空の null 重みで疎密を作る。
var roadsideCatalog = scatterCatalog{
	Zone:    zoneRoadside,
	Density: 0.05,
	Entries: []scatterEntry{
		{Ref: "", Weight: 40}, // 置かない
		{Ref: "barrel", Weight: 12},
		{Ref: "crate", Weight: 12},
		{Ref: "debris", Weight: 10},
		{Ref: "rubble", Weight: 10},
		{Ref: "generator_green", Weight: 4},
		{Ref: "generator_yellow", Weight: 4},
		{Ref: "bench", Weight: 4},
		{Ref: "vending_machine", Weight: 3, Big: true},
		{Ref: "bonfire", Weight: 4, Satellites: []relSpot{{"crate", 1, 1}, {"bench", -1, 0}}},
		{Ref: "forklift", Weight: 3, Big: true, Satellites: []relSpot{{"barrel", 1, 0}, {"debris", -1, 1}}},
	},
}

// wildCatalog は奥地ゾーンの散布定義。自然物を主に密度低め。
var wildCatalog = scatterCatalog{
	Zone:    zoneWild,
	Density: 0.04,
	Entries: []scatterEntry{
		{Ref: "", Weight: 45}, // 置かない
		{Ref: "tree_a", Weight: 14},
		{Ref: "tree_b", Weight: 14},
		{Ref: "plant", Weight: 12},
		{Ref: "moving_stone", Weight: 8},
		{Ref: "big_tree", Weight: 4, Big: true},
		{Ref: "stone_pillar", Weight: 3, Big: true},
	},
}

// scatterCatalogFor は zone の散布定義を返す。zone を1つ足すと switch の網羅を linter が強制する。
func scatterCatalogFor(zone outdoorZone) scatterCatalog {
	switch zone {
	case zoneRoadside:
		return roadsideCatalog
	case zoneWild:
		return wildCatalog
	}
	panic("未知の outdoorZone: " + string(zone))
}

// scatterFeature は wasteland チャンクの開けた地表へ prop を密に散布する feature。
type scatterFeature struct{}

// place は wasteland チャンクだけで density 場を走らせ、選んだタイルへカタログから prop を1個置く。
// 選定はチャンク相対座標と絶対チャンク seed の純関数で、帯の整列がずれても再訪一致する。占有と
// dirt 判定は帯ローカルの実エンティティで引き、経路判定は絶対タイル座標で道の直線と比べる。
func (scatterFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(runSeed, c, rows) != chunkWasteland {
		return nil
	}
	cat := scatterCatalogFor(outdoorZoneAt(runSeed, c, rows))

	tiles := g.tiles.get()
	blocked := blockedTilesInChunk(world, g)

	loX, loY := c.X.Tiles(g.chunkW), c.Y.Tiles(g.chunkH)
	bandLocal := func(rel consts.Coord[consts.Tile]) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: g.offsetX + rel.X, Y: g.offsetY + rel.Y}
	}
	absTile := func(rel consts.Coord[consts.Tile]) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: loX + rel.X, Y: loY + rel.Y}
	}

	// 候補の可否は accept へ委ね、密度場の芯 ScatterArea は候補の意味を知らないまま純関数で保つ。
	accept := func(rel interior.Vec) bool {
		bl := bandLocal(rel)
		return isDirtTile(world, tiles, bl) && !blocked[gc.GridElement{Coord: bl}]
	}

	selSeed := ChunkSeed2D(runSeed^scatterSalt, c.X, c.Y)
	area := interior.Rect{X: 0, Y: 0, W: g.chunkW, H: g.chunkH}
	count := int(math.Round(float64(g.chunkW) * float64(g.chunkH) * cat.Density))

	for _, rel := range interior.ScatterArea(area, accept, selSeed, count) {
		// Big は位相 0 かつ経路外のタイルにだけ乗せる。二段で通行性を担保する
		bigAllowed := onBigPhase(rel) && !onScatterRoute(runSeed, c, rows, g, absTile(rel))
		roll := ChunkSeed2D(selSeed, consts.Chunk(rel.X), consts.Chunk(rel.Y))
		entry := pickScatterEntry(cat.Entries, bigAllowed, roll)
		if entry.Ref == "" {
			continue // null 重み。ここは置かない
		}
		if err := placeScatterEntry(world, tiles, blocked, entry, bandLocal(rel)); err != nil {
			return err
		}
	}
	return nil
}

// placeScatterEntry は anchor へ prop を1個置き、Satellites を anchor 相対へ順に置く。anchor が
// 先の衛星に占有されていれば丸ごと諦める。衛星は dirt 非占有のタイルにだけ置き、収まらない衛星は
// 落とす。anchor だけ置いて衛星を諦める、doc 70 の Satellites と同じ「尽きたら諦める」方針。
func placeScatterEntry(world w.World, tiles map[gc.GridElement]ecs.Entity, blocked map[gc.GridElement]bool, entry scatterEntry, origin consts.Coord[consts.Tile]) error {
	if blocked[gc.GridElement{Coord: origin}] {
		return nil
	}
	if _, err := lifecycle.SpawnProp(world, entry.Ref, origin.X, origin.Y); err != nil {
		return fmt.Errorf("散布 prop の配置に失敗 (%s): %w", entry.Ref, err)
	}
	blocked[gc.GridElement{Coord: origin}] = true
	for _, s := range entry.Satellites {
		pos := consts.Coord[consts.Tile]{X: origin.X + s.dx, Y: origin.Y + s.dy}
		key := gc.GridElement{Coord: pos}
		if blocked[key] || !isDirtTile(world, tiles, pos) {
			continue
		}
		if _, err := lifecycle.SpawnProp(world, s.name, pos.X, pos.Y); err != nil {
			return fmt.Errorf("散布衛星の配置に失敗 (%s): %w", s.name, err)
		}
		blocked[key] = true
	}
	return nil
}

// pickScatterEntry は重み付きで散布物を1つ選ぶ。bigAllowed が偽なら Big の大物を候補から外し、
// 実効的に大物を1位相かつ経路外へ寄せる。Go の map を抽選に使わず slice を順に走って決定的に引く。
func pickScatterEntry(entries []scatterEntry, bigAllowed bool, h uint64) scatterEntry {
	total := 0
	for _, e := range entries {
		if e.Big && !bigAllowed {
			continue
		}
		total += e.Weight
	}
	if total <= 0 {
		return scatterEntry{}
	}
	r := int(h % uint64(total))
	for _, e := range entries {
		if e.Big && !bigAllowed {
			continue
		}
		r -= e.Weight
		if r < 0 {
			return e
		}
	}
	return scatterEntry{}
}

// outdoorZoneAt は wasteland チャンクのゾーンを返す。集落・市街の当選チャンクの近傍を道沿いにし、
// それ以外を奥地にする。道は集落間を結ぶので、集落・市街の近傍が道沿いの帯を兼ねる。近傍地物の
// 位置は既存の道結線と同じく WinnerOf で生成せずに算出する。
func outdoorZoneAt(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) outdoorZone {
	sr := floorDiv(c.X, settlementPlacement.Spacing)
	for _, pr := range []consts.Chunk{sr - 1, sr, sr + 1} {
		if chunkChebyshev(c, settlementPlacement.WinnerOf(runSeed, pr, rows)) <= scatterRoadsideRange {
			return zoneRoadside
		}
	}
	ur := floorDiv(c.X, urbanPlacement.Spacing)
	for _, pr := range []consts.Chunk{ur - 1, ur, ur + 1} {
		if chunkChebyshev(c, urbanPlacement.WinnerOf(runSeed, pr, rows)) <= scatterRoadsideRange {
			return zoneRoadside
		}
	}
	return zoneWild
}

// onBigPhase は相対タイルが大物を乗せる位相かを返す。doc 70 のパリティ格子 (x+y)%2 を一般の P 位相へ
// 広げたもの。相対座標で判定するのでチャンクごとに同じ格子が並び、帯の整列に依らず決定的になる。
// 負の座標でも周期が途切れないよう剰余を床方向へ寄せる。
func onBigPhase(rel consts.Coord[consts.Tile]) bool {
	m := (rel.X + scatterBigPhaseK*rel.Y) % scatterBigPhaseMod
	if m < 0 {
		m += scatterBigPhaseMod
	}
	return m == 0
}

// onScatterRoute は絶対タイル pos が近傍集落間の道の直線からバッファ以内かを返す。実際の道タイルを
// 読まず WinnerOf で算出した集落中心の L 字経路と比べる。道は floor 化済みで散布は dirt にしか
// 置かないので、経路マスクが守るのは舗装路でない wasteland 内の横断レーンである。
func onScatterRoute(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom, pos consts.Coord[consts.Tile]) bool {
	r := floorDiv(c.X, settlementPlacement.Spacing)
	// c を横切りうるのは (r-1, r) と (r, r+1) を結ぶ2本だけ。roadFeature と同じ結線
	for _, pr := range []consts.Chunk{r - 1, r} {
		a := settlementPlacement.WinnerOf(runSeed, pr, rows)
		b := settlementPlacement.WinnerOf(runSeed, pr+1, rows)
		ax := a.X.Tiles(g.chunkW) + g.chunkW/2
		ay := a.Y.Tiles(g.chunkH) + g.chunkH/2
		bx := b.X.Tiles(g.chunkW) + g.chunkW/2
		by := b.Y.Tiles(g.chunkH) + g.chunkH/2
		// 集落 a から b への L 字。水平辺 y=ay と垂直辺 x=bx の近い方までの距離を見る
		d := min(chebToHSeg(pos, ay, ax, bx), chebToVSeg(pos, bx, ay, by))
		if d <= scatterRouteBuffer {
			return true
		}
	}
	return false
}

// blockedTilesInChunk はチャンク g の範囲で BlockPass を持つエンティティの占有タイルを集める。
// 散布はこの占有を避けて dirt の空きにだけ置く。フィーチャ間で共有される占有索引は持たない。
func blockedTilesInChunk(world w.World, g chunkGeom) map[gc.GridElement]bool {
	blocked := make(map[gc.GridElement]bool)
	loX, hiX := g.offsetX, g.offsetX+g.chunkW
	loY, hiY := g.offsetY, g.offsetY+g.chunkH
	q := query.ActiveFilter2[gc.GridElement, gc.BlockPass](world).Query()
	for q.Next() {
		ge := *world.Components.GridElement.Get(q.Entity())
		if ge.X >= loX && ge.X < hiX && ge.Y >= loY && ge.Y < hiY {
			blocked[ge] = true
		}
	}
	return blocked
}

// isDirtTile は帯ローカル座標 pos のタイルが土かを返す。道の floor や他フィーチャの生成物は土でない
// ので散布から外れる。road.go の replaceDirtTile と同じ土判定。
func isDirtTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile]) bool {
	e, ok := tiles[gc.GridElement{Coord: pos}]
	if !ok || !world.ECS.Alive(e) || !world.Components.Name.Has(e) {
		return false
	}
	return world.Components.Name.Get(e).Name == consts.TileNameDirt
}

// chunkChebyshev は2チャンク座標のチェビシェフ距離を返す。ゾーン判定の近接度に使う。
func chunkChebyshev(a, b consts.Coord[consts.Chunk]) consts.Chunk {
	return max(absInt(a.X-b.X), absInt(a.Y-b.Y))
}

// chebToHSeg は点 p から水平線分、y=segY で x が [x0, x1]、までのチェビシェフ距離を返す。
func chebToHSeg(p consts.Coord[consts.Tile], segY, x0, x1 consts.Tile) consts.Tile {
	lo, hi := min(x0, x1), max(x0, x1)
	var dx consts.Tile
	if p.X < lo {
		dx = lo - p.X
	} else if p.X > hi {
		dx = p.X - hi
	}
	return max(dx, absInt(p.Y-segY))
}

// chebToVSeg は点 p から垂直線分、x=segX で y が [y0, y1]、までのチェビシェフ距離を返す。
func chebToVSeg(p consts.Coord[consts.Tile], segX, y0, y1 consts.Tile) consts.Tile {
	lo, hi := min(y0, y1), max(y0, y1)
	var dy consts.Tile
	if p.Y < lo {
		dy = lo - p.Y
	} else if p.Y > hi {
		dy = p.Y - hi
	}
	return max(dy, absInt(p.X-segX))
}

// absInt は Tile と Chunk の双方で使う絶対値。どちらも基底が int なので型引数で共有する。
func absInt[T ~int](v T) T {
	if v < 0 {
		return -v
	}
	return v
}
