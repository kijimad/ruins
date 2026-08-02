package overworld

import (
	"fmt"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 屋外の散布は wasteland チャンクの開けた地表を自然の野原に仕立てる feature。草・低木・岩を撒く。
// 草は透明背景の prop として地面に重ねる。タイルのオートタイルで草地を作ると、
// 同種で埋めた面は中央が暗いフィルに、異種の境界は縁取りになり自然に繋がらないためである。prop なら
// オートタイルを経由せず、地面へそのまま重なる。疎な Placement と違い、密度場と density 確率で個数を
// 決める。他フィーチャの後に評価し、道が floor 化した後の実状態を読んで土系の地面かつ非占有のタイルだけに
// 置く。人工物を原野一面に撒くと不自然なので置かない。

// scatterEntry は散布物の1候補。実スプライトのある自然物を使う。Big は通行を塞ぐ大物の印で、
// 位相格子と経路マスクの対象になる。Ref が空文字なら「置かない」を表し、null 重みとして扱う。
// Satellites があれば anchor 相対の小クラスタを1回で置く。
type scatterEntry struct {
	Ref        string
	Weight     int
	Big        bool
	Satellites []relSpot
}

// scatterCatalog は zone ごとの散布定義。GrassDensity は草・雑草 prop を撒く面積あたり係数、PropDensity
// は樹木・岩を撒く係数で、どちらも count = round(area * Density) で個数を密度確率から導く。個数を反復
// 乱数でなく密度確率で決めるので純関数のまま。
type scatterCatalog struct {
	Zone         outdoorZone
	GrassDensity float64
	PropDensity  float64
	Entries      []scatterEntry
}

// outdoorZone は wasteland チャンクの人の手の残り具合。近傍の集落・市街地までの距離で決まる。
// 実体は文字列にし、%v やログで数値でなく種別名が出て読みやすい。iota の連番は割に合わないので使わない。
type outdoorZone string

const (
	// zoneRoadside は道・集落・市街に近い。人が踏み分けた開けた原で、草や低木は疎。
	zoneRoadside outdoorZone = "roadside"
	// zoneWild は奥地。草が濃く、低木・木立・岩が茂る自然の野原。
	zoneWild outdoorZone = "wild"
)

// 草の prop の名前と混ぜ方を決める。透明 prop なので地面へ重なり、オートタイルの縁や暗いフィルが出ない。
// 大半をみずみずしい草にし、乾いた雑草をまばらに混ぜて単調さを崩す。
const (
	scatterGrassProp = "grass"
	scatterWeedProp  = "weeds"
	// scatterWeedMod は雑草の割合。hash がこの法で 0 になるタイルだけ雑草にする。約 1/4。
	scatterWeedMod = 4
)

// 位相格子と経路マスクの定数。どちらも Big の大物だけに効き、通行を塞がない小物は制約しない。
const (
	// scatterBigPhaseMod は大物を乗せる位相の法。候補タイルを (x + scatterBigPhaseK*y) mod P へ分け、
	// Big は位相 0 のタイルにだけ乗せる。実効密度が 1/P に希釈され、P が通行性のダイヤルになる。
	scatterBigPhaseMod consts.Tile = 3
	scatterBigPhaseK   consts.Tile = 2
	// scatterRouteBuffer は経路マスクのバッファ幅。隊商の横断レーンを確保するため、算出した経路の
	// 直線からこのタイル数以内では Big の大物を置かない。road.go の roadWidth とは独立で、舗装済みの道は
	// floor 化されて isEarthTile で弾かれ散布対象外になる。ここが守るのは舗装されない wasteland 内の横断レーン。
	scatterRouteBuffer consts.Tile = 2
	// scatterRoadsideRange は道沿いゾーンの半径。集落・市街の当選チャンクからこのチャンク数以内を
	// 道沿いにする。道は集落間を結ぶので、集落の近傍が道沿いの帯を兼ねる。
	scatterRoadsideRange consts.Chunk = 1
	// scatterGrassChannel は草の選定を prop 選定と無相関にするハッシュチャネル。
	scatterGrassChannel uint64 = 0x67726173735f7072 // "grass_pr"
	// scatterWeedChannel は雑草の抽選を草の選定と無相関にするハッシュチャネル。
	scatterWeedChannel uint64 = 0x776565645f5f7072 // "weed__pr"
)

// scatterEarthTiles は自然物を立てられる土系タイル。草の prop もこの上に撒く。dirt が基調で、砂土は
// 将来ムラを足すときの受け皿として許す。
var scatterEarthTiles = map[string]bool{
	consts.TileNameDirt: true, "sand_orange": true, "sand_red": true, "sand_pink": true,
}

// roadsideCatalog は道沿いゾーンの散布定義。踏み分けられた原なので草も低木もまばらにする。
var roadsideCatalog = scatterCatalog{
	Zone:         zoneRoadside,
	GrassDensity: 0.18,
	PropDensity:  0.015,
	Entries: []scatterEntry{
		{Ref: "", Weight: 55}, // 置かない
		{Ref: "tree_a", Weight: 16},
		{Ref: "tree_b", Weight: 13},
		{Ref: "rock", Weight: 10},
	},
}

// wildCatalog は奥地ゾーンの散布定義。草が濃く、低木が茂り、時折木立が立つ。
var wildCatalog = scatterCatalog{
	Zone:         zoneWild,
	GrassDensity: 0.38,
	PropDensity:  0.035,
	Entries: []scatterEntry{
		{Ref: "", Weight: 38}, // 置かない
		{Ref: "tree_a", Weight: 24},
		{Ref: "tree_b", Weight: 24},
		{Ref: "rock", Weight: 14},
		// 木立。大木の周りに低木が寄り添う小クラスタ
		{Ref: "big_tree", Weight: 6, Big: true, Satellites: []relSpot{{"tree_a", 1, 0}, {"tree_b", -1, 1}}},
	},
}

// scatterCatalogFor は zone の散布定義を返す。exhaustive linter は iota 整数だけでなく、同一 package に
// 型付き定数を持つ named 型を enum とみなすので、string の outdoorZone も網羅検査の対象になる。default を
// 置かなければ zone を1つ足したとき case 漏れを lint が止める。.golangci.yml の
// default-signifies-exhaustive=true が前提で、chunkType など既存の string enum も同じ方式に依る。実際に
// case を1つ消すと「missing cases in switch of type outdoorZone」で lint が落ちることを確認済み。末尾
// panic は網羅漏れ防止に加え、未知 zone のランタイム保護も兼ねる。
func scatterCatalogFor(zone outdoorZone) scatterCatalog {
	switch zone {
	case zoneRoadside:
		return roadsideCatalog
	case zoneWild:
		return wildCatalog
	}
	panic("未知の outdoorZone: " + string(zone))
}

// scatterFeature は wasteland チャンクの開けた地表へ草・低木・岩を散布する feature。
type scatterFeature struct{}

// place は wasteland チャンクだけで密度場を走らせ、まず草を密に撒き、続けて樹木と岩を疎に撒く。
// 選定はチャンク相対座標と絶対チャンク seed の純関数で、帯の整列がずれても再訪一致する。地面判定と
// 占有は帯ローカルの実エンティティで引き、経路判定は絶対タイル座標で道の直線と比べる。
func (scatterFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(runSeed, c, rows) != chunkWasteland {
		return nil
	}
	cat := scatterCatalogFor(outdoorZoneAt(runSeed, c, rows))

	tiles := g.tiles.get()
	occupied := blockedTilesInChunk(world, g)

	loX, loY := c.X.Tiles(g.chunkW), c.Y.Tiles(g.chunkH)
	bandLocal := func(rel consts.Coord[consts.Tile]) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: g.offsetX + rel.X, Y: g.offsetY + rel.Y}
	}
	absTile := func(rel consts.Coord[consts.Tile]) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: loX + rel.X, Y: loY + rel.Y}
	}
	area := interior.Rect{X: 0, Y: 0, W: g.chunkW, H: g.chunkH}
	// 候補は土系の地面かつ非占有。占有は BlockPass の壁と、先に撒いた草・prop を含む
	accept := func(rel interior.Vec) bool {
		bl := bandLocal(rel)
		return isEarthTile(world, tiles, bl) && !occupied[gc.GridElement{Coord: bl}]
	}

	// まず草・雑草を密に撒く。透明 prop なのでオートタイルの縁が出ず、地面へ自然に重なる。歩行可能なので
	// 通行を塞がず、位相や経路の制約は要らない。続く樹木が草と重ならないよう占有を記録しながら先に撒く。
	grassSeed := ChunkSeed2D(runSeed^scatterSalt^scatterGrassChannel, c.X, c.Y)
	grassCount := int(math.Round(float64(g.chunkW) * float64(g.chunkH) * cat.GrassDensity))
	for _, rel := range interior.ScatterArea(area, accept, grassSeed, grassCount) {
		bl := bandLocal(rel)
		name := scatterGrassProp
		if hashTileCoord(grassSeed^scatterWeedChannel, rel)%scatterWeedMod == 0 {
			name = scatterWeedProp
		}
		if _, err := lifecycle.SpawnProp(world, name, bl.X, bl.Y); err != nil {
			return fmt.Errorf("草の配置に失敗 (%s): %w", name, err)
		}
		occupied[gc.GridElement{Coord: bl}] = true
	}

	// 続けて樹木・岩を疎に撒く。Big の大物は位相 0 かつ経路外のタイルにだけ乗せ、二段で通行性を担保する。
	selSeed := ChunkSeed2D(runSeed^scatterSalt, c.X, c.Y)
	propCount := int(math.Round(float64(g.chunkW) * float64(g.chunkH) * cat.PropDensity))
	for _, rel := range interior.ScatterArea(area, accept, selSeed, propCount) {
		bigAllowed := onBigPhase(rel) && !onScatterRoute(runSeed, c, rows, g, absTile(rel))
		roll := hashTileCoord(selSeed, rel)
		entry := pickScatterEntry(cat.Entries, bigAllowed, roll)
		if entry.Ref == "" {
			continue // null 重み。ここは置かない
		}
		if err := placeScatterEntry(world, tiles, occupied, entry, bandLocal(rel)); err != nil {
			return err
		}
	}
	return nil
}

// placeScatterEntry は anchor へ prop を1個置き、Satellites を anchor 相対へ順に置く。anchor が
// 先に占有されていれば丸ごと諦める。衛星は土系の地面かつ非占有のタイルにだけ置き、収まらない衛星は
// 落とす。anchor だけ置いて衛星を諦める「尽きたら諦める」方針にする。
func placeScatterEntry(world w.World, tiles map[gc.GridElement]ecs.Entity, occupied map[gc.GridElement]bool, entry scatterEntry, origin consts.Coord[consts.Tile]) error {
	// ScatterArea は選定時点の occupied で候補を絞るが、その後この関数が衛星を置いて occupied を更新する。
	// 同じ ScatterArea が返した後続の anchor が、先の entry の衛星に占有されうるので入口で再確認する。
	if occupied[gc.GridElement{Coord: origin}] {
		return nil
	}
	if _, err := lifecycle.SpawnProp(world, entry.Ref, origin.X, origin.Y); err != nil {
		return fmt.Errorf("散布 prop の配置に失敗 (%s): %w", entry.Ref, err)
	}
	occupied[gc.GridElement{Coord: origin}] = true
	for _, s := range entry.Satellites {
		pos := consts.Coord[consts.Tile]{X: origin.X + s.dx, Y: origin.Y + s.dy}
		key := gc.GridElement{Coord: pos}
		if occupied[key] || !isEarthTile(world, tiles, pos) {
			continue
		}
		if _, err := lifecycle.SpawnProp(world, s.name, pos.X, pos.Y); err != nil {
			return fmt.Errorf("散布衛星の配置に失敗 (%s): %w", s.name, err)
		}
		occupied[key] = true
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
	// total は候補の重み合計で r は [0, total) なので、候補の重みを引き切る前に必ず r < 0 になる。
	// ここへ来るのは重み合計の計算と減算がずれたときだけで、内部データの不変条件違反にあたる
	panic("pickScatterEntry: 重み抽選が候補を引けなかった")
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

// onBigPhase は相対タイルが大物を乗せる位相かを返す。パリティ格子 (x+y)%2 を一般の P 位相へ広げたもの。
// 相対座標で判定するのでチャンクごとに同じ格子が並び、帯の整列に依らず決定的になる。
// 負の座標でも周期が途切れないよう剰余を床方向へ寄せる。
func onBigPhase(rel consts.Coord[consts.Tile]) bool {
	m := (rel.X + scatterBigPhaseK*rel.Y) % scatterBigPhaseMod
	if m < 0 {
		m += scatterBigPhaseMod
	}
	return m == 0
}

// onScatterRoute は絶対タイル pos が近傍集落間の道の直線からバッファ以内かを返す。実際の道タイルを
// 読まず WinnerOf で算出した集落中心の L 字経路と比べる。道は floor 化済みで散布は土にしか置かないので、
// 経路マスクが守るのは舗装路でない wasteland 内の横断レーンである。
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
// 散布はこの占有を避けて空きにだけ置く。フィーチャ間で共有される占有索引は持たない。
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

// isEarthTile は帯ローカル座標 pos が土系タイル、dirt か砂土、かを返す。道の floor や壁、他フィーチャの
// 生成物は土系でないので散布から外れる。
func isEarthTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile]) bool {
	return scatterEarthTiles[tileNameAt(world, tiles, pos)]
}

// hashTileCoord は seed とタイル座標から決定的な 64bit を返す。ChunkSeed2D は2次元整数の純ハッシュ
// なのでタイル座標にも使えるが、引数型が consts.Chunk なのでキャストをここへ隔離し、呼び出し側の意味を
// 「タイル単位の抽選」に保つ。散布物やアクセントの per-tile 抽選に使う。
func hashTileCoord(seed uint64, p consts.Coord[consts.Tile]) uint64 {
	return ChunkSeed2D(seed, consts.Chunk(p.X), consts.Chunk(p.Y))
}

// tileNameAt は帯ローカル座標 pos のタイル名を返す。タイルが無ければ空文字。
func tileNameAt(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile]) string {
	e, ok := tiles[gc.GridElement{Coord: pos}]
	if !ok || !world.ECS.Alive(e) || !world.Components.Name.Has(e) {
		return ""
	}
	return world.Components.Name.Get(e).Name
}

// chunkChebyshev は2チャンク座標のチェビシェフ距離を返す。ゾーン判定の近接度に使う。
func chunkChebyshev(a, b consts.Coord[consts.Chunk]) consts.Chunk {
	return max(geometry.Abs(a.X-b.X), geometry.Abs(a.Y-b.Y))
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
	return max(dx, geometry.Abs(p.Y-segY))
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
	return max(dy, geometry.Abs(p.X-segX))
}
