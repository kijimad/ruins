package overworld

import (
	"fmt"
	"math/rand/v2"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 市街地は区域の1種類で、建物チャンクの2次元格子。1チャンク = 1建物とし、街路で区切る。
// 各チャンクは自分の建物を、区域の seed と区域の中の座標から独立に決める。
// 街路は各チャンクの北辺・西辺に敷き、隣接チャンクと連続して格子状の街並みになる。
const urbanStreetW consts.Tile = 4 // チャンクの北辺・西辺の街路の幅。2車線+歩道ぶん

// urbanFeature は市街地の feature 実装。
type urbanFeature struct{}

// urbanEnemyTableFor は施設行の enemyTable が指す敵テーブルを返す。
func urbanEnemyTableFor(raws rawschema.Raws, fac string) (rawschema.EnemyTable, error) {
	f, ok := raw.GetFacility(raws, fac)
	if !ok {
		return rawschema.EnemyTable{}, fmt.Errorf("urban enemy table: facility %q not found", fac)
	}
	et, err := raw.GetEnemyTable(raws, f.EnemyTable)
	if err != nil {
		return rawschema.EnemyTable{}, fmt.Errorf("urban enemy table for facility %q: %w", fac, err)
	}
	return et, nil
}

// facilityWeight は施設の抽選重みと規模 gate。市街地の一辺が minSpan 以上のときだけ抽選対象になる。
type facilityWeight struct {
	kind    string
	weight  int
	minSpan consts.Chunk
}

// zoneCatalog は地区ごとの施設抽選重み。
type zoneCatalog map[rawschema.Zone][]facilityWeight

// zoneCatalogFrom は facilities 行から地区ごとの施設抽選重みを導出する。
func zoneCatalogFrom(raws rawschema.Raws) zoneCatalog {
	cat := make(zoneCatalog)
	for _, f := range raw.PtrSlice(raws.Facilities) {
		for _, z := range f.Zones {
			cat[z.Zone] = append(cat[z.Zone], facilityWeight{kind: f.Id, weight: int(z.Weight), minSpan: consts.Chunk(z.MinSpan)})
		}
	}
	return cat
}

// zoneChoiceShift は層の候補から地区を選ぶ、区域の seed のビットの位置
const zoneChoiceShift = 32

// zoneAt は区域の中の座標 l の地区を返す。並べ方で層を決め、層の候補から区域の seed で1つ選ぶ
func (r districtRegion) zoneAt(l consts.Coord[consts.Chunk]) rawschema.Zone {
	zones := r.kind.Layers[r.layerAt(l)].Zones
	return zones[(r.seed>>zoneChoiceShift)%uint64(len(zones))]
}

// layerAt は区域の中の座標 l が属する層の添字を、区域の並べ方に従って返す
func (r districtRegion) layerAt(l consts.Coord[consts.Chunk]) int {
	switch r.kind.Pattern {
	case rawschema.DistrictPatternRings:
		if inCore(l.X, l.Y, r.w, r.h) {
			return 1
		}
		return 0
	case rawschema.DistrictPatternRows:
		return rowsLayer(r.kind.Layers, l.Y, r.h)
	}
	// raw の列挙の外の並べ方は1つ目の層にする
	return 0
}

// inCore は区域の中の座標が中心の層に入るかを返す。中心からの2倍座標のチェビシェフ距離が、短い辺から3を引いた値以下なら中心
func inCore(lx, ly, cw, ch consts.Chunk) bool {
	dx := int(lx)*2 - int(cw-1)
	dy := int(ly)*2 - int(ch-1)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return max(dx, dy) <= int(min(cw, ch))-3
}

// rowsLayer は高さ ch の区域の行 ly が属する層の添字を返す。行の数を省いた層より前の層は南の端から、
// 後ろの層は北の端から順に行を占め、残りの行を行の数を省いた層が占める
func rowsLayer(layers []rawschema.DistrictLayer, ly, ch consts.Chunk) int {
	rest := slices.IndexFunc(layers, func(l rawschema.DistrictLayer) bool { return l.Rows == nil })
	fromSouth := int(ch - 1 - ly)
	for i := range rest {
		if fromSouth < *layers[i].Rows {
			return i
		}
		fromSouth -= *layers[i].Rows
	}
	fromNorth := int(ly)
	for i := len(layers) - 1; i > rest; i-- {
		if fromNorth < *layers[i].Rows {
			return i
		}
		fromNorth -= *layers[i].Rows
	}
	return rest
}

// rollFacilityInZone は地区の重み表から規模 gate を通った施設 id を重みで抽選する。exclude の施設は引かない
func rollFacilityInZone(rng *rand.Rand, cat zoneCatalog, z rawschema.Zone, span consts.Chunk, exclude string) string {
	zc := cat[z]
	i, ok := consts.Pick(rng, zc, func(f facilityWeight) float64 {
		if span < f.minSpan || f.kind == exclude {
			return 0
		}
		return float64(f.weight)
	})
	if !ok {
		// 一番小さい市街地でも各地区に建てられる基本施設があることを raw の検証が保証する
		panic("rollFacilityInZone: zone has no facility within span")
	}
	return zc[i].kind
}

// urbanFacilityAt は区域 r の中のチャンク c に建つ施設 id を返す。地図表示と生成の両方が使う。
func urbanFacilityAt(raws rawschema.Raws, cat zoneCatalog, r districtRegion, c consts.Coord[consts.Chunk]) string {
	return lotAt(lotsOf(raws, cat, r), r.local(c)).facility
}

// place は c が市街地の建物チャンクなら自分の建物を描く。各チャンクは自己完結するので
// 生成順に依存しない。
func (urbanFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	raws := world.Resources.RawMaster
	r, ok := districtAt(raws, runSeed, c, cols)
	if !ok {
		return nil
	}
	l := r.local(c)
	lot := lotAt(lotsOf(raws, zoneCatalogFrom(raws), r), l)
	var fence []consts.Coord[consts.Tile]
	if r.kind.Fence != nil && *r.kind.Fence {
		north, south := gateColumns(raws, r)
		fence = districtFence(r, l, g.chunkW, g.chunkH, north-r.anchor.X, south-r.anchor.X)
	}
	verge := r.kind.Verge != nil && *r.kind.Verge
	return renderUrbanChunk(world, g, ChunkSeed2D(r.seed, l.X, l.Y), r.span(), lot, l.Sub(lot.anchor), urbanDangerAt(world, c), fence, verge)
}

// gateColumns は塀を巡らせる区域の、北と南の門を開ける絶対チャンク列を返す。真ん中の列を覆う区画の西の端の列で、
// 区画の西の帯は街路なので、門から区域の中の街路へ入れる
func gateColumns(raws rawschema.Raws, r districtRegion) (north, south consts.Chunk) {
	lots := lotsOf(raws, zoneCatalogFrom(raws), r)
	north = lotAt(lots, consts.Coord[consts.Chunk]{X: r.w / 2, Y: 0}).anchor.X
	south = lotAt(lots, consts.Coord[consts.Chunk]{X: r.w / 2, Y: r.h - 1}).anchor.X
	return r.anchor.X + north, r.anchor.X + south
}

// districtFence は区域の中のチャンク l に置く塀のマスを、チャンク内の座標で返す。塀は区域の外周を巡り、
// 区域の中の列 northGate と southGate の北と南の端に、街路の幅だけ門を開ける。東と南の辺には街路が無いので、
// 建物の区画の外側の1列に置く
func districtFence(r districtRegion, l consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile, northGate, southGate consts.Chunk) []consts.Coord[consts.Tile] {
	northOpen := l.X == northGate && l.Y == 0
	southOpen := l.X == southGate && l.Y == r.h-1
	var out []consts.Coord[consts.Tile]
	add := func(x, y consts.Tile) {
		if ((northOpen && y == 0) || (southOpen && y == chunkH-1)) && x < urbanStreetW {
			return
		}
		out = append(out, consts.Coord[consts.Tile]{X: x, Y: y})
	}
	for y := range chunkH {
		for x := range chunkW {
			west := x == 0 && l.X == 0
			east := x == chunkW-1 && l.X == r.w-1
			north := y == 0 && l.Y == 0
			south := y == chunkH-1 && l.Y == r.h-1
			if west || east || north || south {
				add(x, y)
			}
		}
	}
	return out
}

// urbanDangerAt はチャンクの敵抽選に使う危険度を返す。北進度(空間)と日数(時間)の高い方。
// c.Y は絶対チャンク行なので DepthOfChunkRow へそのまま渡せ、生成の再訪一致を壊さない。
func urbanDangerAt(world w.World, c consts.Coord[consts.Chunk]) consts.Danger {
	danger := query.DangerLevelAt(world)
	if sb := query.GetSeamlessBand(world); sb != nil {
		if d := query.DangerLevelForDepth(sb.DepthOfChunkRow(c.Y)); d > danger {
			danger = d
		}
	}
	return danger
}

// renderUrbanChunk は1チャンクに街路と施設の絵と塀を置き、規模に応じた敵を湧かせる。塀はチャンク内の座標で受け、
// verge なら街路の両端を植え込みにする。
// 部屋の絵を持つ施設は区画の中の座標 l の部屋と帯を組み、それ以外は1棟の絵を描く
func renderUrbanChunk(world w.World, g chunkGeom, seed uint64, size consts.Chunk, lot buildingLot, l consts.Coord[consts.Chunk], danger consts.Danger, fence []consts.Coord[consts.Tile], verge bool) error {
	// ストリーム識別子 0x2 は建物幾何と敵配置。施設抽選の 0x1、施設の絵の 0x3 と分けて相互干渉を避ける
	rng := rand.New(rand.NewPCG(seed, 0x2))
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return fmt.Errorf("failed to load facility templates: %w", err)
	}
	cells, origin, indoor, err := planUrbanChunk(world.Resources.RawMaster, loader, lot, l, g.chunkW, g.chunkH, seed, rng, verge)
	if err != nil {
		return err
	}
	if origin != 0 {
		if err := drawUrbanStreets(world, g, verge); err != nil {
			return err
		}
	}
	isWall, occupied, err := spawnFacility(world, g, cells, origin, origin, lot.facility, danger, seed, indoor)
	if err != nil {
		return err
	}
	fac := lot.facility
	for _, p := range fence {
		pos := consts.Coord[consts.Tile]{X: g.offsetX + p.X, Y: g.offsetY + p.Y}
		occupied[pos] = true
		// 建物の外壁が敷地の外周に来れば、外壁を塀の代わりにする
		if isWall(p.X, p.Y) || slices.ContainsFunc(query.GetEntitiesAt(world, pos.X, pos.Y), world.Components.Prop.Has) {
			continue
		}
		if _, err := lifecycle.SpawnProp(world, "fence", pos.X, pos.Y); err != nil {
			return fmt.Errorf("failed to place district fence at %d,%d: %w", pos.X, pos.Y, err)
		}
	}
	return spawnUrbanEnemies(world, g, rng, size, fac, danger, isWall, occupied)
}

// planUrbanChunk は区画 lot の中の座標 l のチャンクに置くマスと、マスを置くチャンク内の原点を返す。部屋の絵を持つ施設は
// 部屋と帯でチャンク全体を組み、それ以外は街路の帯の内側に1棟の絵を置く。厚い壁の塊の内側は虚空にする
func planUrbanChunk(raws rawschema.Raws, loader *maptemplate.TemplateLoader, lot buildingLot, l consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile, seed uint64, rng *rand.Rand, verge bool) ([][]maptemplate.MapCell, consts.Tile, func(x, y int) bool, error) {
	if loader.HasRooms(lot.facility) {
		cells, indoor, err := roomChunkCells(loader, lot, l, chunkW, chunkH, verge)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to plan facility %q room: %w", lot.facility, err)
		}
		// チャンクの北と西の外、区画の内側の東と南の外は隣の部屋の壁か扉の帯なので壁と数える。区画の東と南の外は隣の区画の街路
		w, h := int(chunkW), int(chunkH)
		voidEnclosedWalls(cells, tileBlocksPass(raws), func(x, y int) bool {
			return x >= w && l.X == lot.w-1 || y >= h && l.Y == lot.h-1
		})
		return cells, 0, indoor, nil
	}
	cells, err := planFacility(loader, lot.facility, rand.New(rand.NewPCG(seed, 0x3)).Uint64(), rng)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to plan facility %q: %w", lot.facility, err)
	}
	// 1棟の絵は街路の帯に囲まれるので、絵の外は通れるものとして数える
	voidEnclosedWalls(cells, tileBlocksPass(raws), func(int, int) bool { return true })
	return cells, urbanStreetW, nil, nil
}

// drawUrbanStreets はチャンクの北辺と西辺に街路を描く。建物区画のタイルは建物の生成が描くので触らない
func drawUrbanStreets(world w.World, g chunkGeom, verge bool) error {
	tiles := g.tiles.get()
	for ly := range g.chunkH {
		for lx := range g.chunkW {
			if lx >= urbanStreetW && ly >= urbanStreetW {
				continue
			}
			if _, err := replaceTile(world, tiles, consts.Coord[consts.Tile]{X: g.offsetX + lx, Y: g.offsetY + ly}, streetTerrain(lx, ly, verge)); err != nil {
				return fmt.Errorf("failed to place urban street (x=%d, y=%d): %w", g.offsetX+lx, g.offsetY+ly, err)
			}
		}
	}
	return nil
}

// streetVergeTile は街路の植え込みの地形
const streetVergeTile = "grass_green"

// streetTerrain は街路の帯のマス (x, y) の地形を返す。verge なら帯の両端の1マスを植え込みにして、建物と道の間を空ける。
// 北と西の帯が交わる角は全て道にする
func streetTerrain(x, y consts.Tile, verge bool) string {
	edge := func(v consts.Tile) bool { return v == 0 || v == urbanStreetW-1 }
	switch {
	case !verge || x < urbanStreetW && y < urbanStreetW:
		return consts.TileNameFloor
	case y < urbanStreetW && edge(y), x < urbanStreetW && edge(x):
		return streetVergeTile
	}
	return consts.TileNameFloor
}

// spawnUrbanEnemies はチャンクに敵を数体湧かせる。数は市街地の規模に比例し、種類は敵テーブルから
// 危険度で重み抽選する。壁マスに埋まる位置は避ける。危険度は呼び出し側が北進度と日数から決める。
func spawnUrbanEnemies(world w.World, g chunkGeom, rng *rand.Rand, size consts.Chunk, fac string, danger consts.Danger, isWall func(lx, ly consts.Tile) bool, occupied map[consts.Coord[consts.Tile]]bool) error {
	enemyTable, err := urbanEnemyTableFor(world.Resources.RawMaster, fac)
	if err != nil {
		return fmt.Errorf("failed to get urban enemy table: %w", err)
	}
	// 湧き数は街の規模で決める。敵種別は危険度でフィルタし、北へ進むほど・日が進むほど強敵が出る。
	count := 1 + rng.IntN(int(size))
	for range count {
		enemyName, err := raw.SelectEnemyByWeight(enemyTable, rng, int(danger))
		if err != nil {
			return fmt.Errorf("failed to select urban enemy: %w", err)
		}
		if enemyName == "" {
			continue // 危険度帯に該当する敵がいなければこの1体は諦める
		}
		// 敵は街路に湧かせ、建物の footprint と壁の上は避ける。占有は footprint 全域に及ぶので、一度引いて
		// 塞がっていたら目標数を満たすよう空きが出るまで位置を引き直す。試行を尽くしても空かなければその
		// 1体は諦める。抽選順は固定なので同じ seed なら同じ結果になる
		for range urbanSpawnTries {
			lx := consts.Tile(rng.IntN(int(g.chunkW)))
			ly := consts.Tile(rng.IntN(int(g.chunkH)))
			pos := consts.Coord[consts.Tile]{X: g.offsetX + lx, Y: g.offsetY + ly}
			if isWall(lx, ly) || occupied[pos] {
				continue
			}
			if _, err := lifecycle.SpawnEnemy(world, pos, enemyName); err != nil {
				return fmt.Errorf("failed to place urban enemy: %w", err)
			}
			break
		}
	}
	return nil
}

// urbanSpawnTries は敵1体あたり街路の空きタイルを探す最大試行数。建物が footprint 全域を占有し有効タイルが
// 減るので、数回の引き直しで目標数をほぼ満たせるだけの余裕を持たせる。
const urbanSpawnTries = 24

// tileIndex はチャンク生成中に地物が共有するタイルの座標引き索引。地物が壁や道を置換する
// とき使う。最初に必要とした地物が全域スキャンで構築し、以降の地物は再利用する。壁を置かない
// 荒れ地では一度も構築されず、全域スキャンの無駄を避ける。replaceTile が置換後の実体を書き
// 戻すので、地物をまたいでも索引は実状態を映し続け、二重残留を防ぐ。
type tileIndex struct {
	world    w.World
	loX, hiX consts.Tile
	tiles    map[gc.GridElement]ecs.Entity
}

// get は索引を返す。未構築なら遅延構築する。壁を置く地物が現れて初めてスキャンが走る。
func (ix *tileIndex) get() map[gc.GridElement]ecs.Entity {
	if ix.tiles == nil {
		ix.tiles = tileEntitiesInRange(ix.world, ix.loX, ix.hiX)
	}
	return ix.tiles
}

// tileEntitiesInRange は X 範囲 [loX, hiX) のタイルエンティティを座標引きできるよう集める。
func tileEntitiesInRange(world w.World, loX, hiX consts.Tile) map[gc.GridElement]ecs.Entity {
	tiles := make(map[gc.GridElement]ecs.Entity)
	q := query.ActiveFilter2[gc.GridElement, gc.Tile](world).Query()
	for q.Next() {
		e := q.Entity()
		g := *world.Components.GridElement.Get(e)
		if g.X >= loX && g.X < hiX {
			tiles[g] = e
		}
	}
	return tiles
}

// replaceTile は座標のタイルを取り除き、tileName のタイルへ置き換える。
// オートタイル添字は仮の 0 で置き、後段の RecalcAutotileInXRange が実状態から揃える。
func replaceTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile], tileName string) (ecs.Entity, error) {
	return replaceTileIndexed(world, tiles, pos, tileName, new(0))
}

// replaceTileIndexed は replaceTile と同じく置き換え、オートタイル添字 index を渡す。オートタイルしないタイルは nil を渡す
func replaceTileIndexed(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile], tileName string, index *int) (ecs.Entity, error) {
	g := gc.GridElement{Coord: pos}
	if e, ok := tiles[g]; ok && world.ECS.Alive(e) {
		world.ECS.RemoveEntity(e)
	}
	e, err := lifecycle.SpawnTile(world, tileName, pos.X, pos.Y, index)
	if err != nil {
		return ecs.Entity{}, err
	}
	// 置換後の実体を索引へ書き戻す。索引を地物間で共有するため、後続の地物が同じ座標を
	// 正しく再置換でき、旧タイルの二重残留を防ぐ。呼び出し側が実体へ属性を足せるよう返す
	tiles[g] = e
	return e, nil
}
