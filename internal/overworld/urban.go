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

// industrialUrbanBit は市街地の性格を住宅地寄りか産業区寄りかに振る、区域の seed のビット。
const industrialUrbanBit = uint64(1) << 32

// zoneOf は市街地内のローカルチャンク座標から地区を決める純関数。中心からの2倍座標のチェビシェフ距離が
// 短い辺から3を引いた値以下なら都心、それ以外は市街地の性格で住宅地か産業区にする
func zoneOf(lx, ly, cw, ch consts.Chunk, urbanSeed uint64) rawschema.Zone {
	dx := int(lx)*2 - int(cw-1)
	dy := int(ly)*2 - int(ch-1)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if max(dx, dy) <= int(min(cw, ch))-3 {
		return rawschema.Downtown
	}
	if urbanSeed&industrialUrbanBit != 0 {
		return rawschema.Industrial
	}
	return rawschema.Residential
}

// campusZoneOf は研究所の敷地の中の行 ly から地区を決める。南の端の列を門、北の端の列を奥の閉鎖区画、
// 残りを中ほどにする。プレイヤーは南から北へ進むので、門から入って奥ほど危険になる
func campusZoneOf(ly, ch consts.Chunk) rawschema.Zone {
	switch ly {
	case ch - 1:
		return rawschema.CampusGate
	case 0:
		return rawschema.CampusQuarantine
	default:
		return rawschema.CampusInner
	}
}

// zoneAt は区域の中の座標 l の地区を、区域の作りに従って返す
func (r districtRegion) zoneAt(l consts.Coord[consts.Chunk]) rawschema.Zone {
	switch r.layout {
	case rawschema.DistrictLayoutTown:
		return zoneOf(l.X, l.Y, r.w, r.h, r.seed)
	case rawschema.DistrictLayoutCampus:
		return campusZoneOf(l.Y, r.h)
	}
	// raw の列挙の外の作りは町として扱う
	return zoneOf(l.X, l.Y, r.w, r.h, r.seed)
}

// rollFacilityInZone は地区の重み表から規模 gate を通った施設 id を重みで抽選する。
func rollFacilityInZone(rng *rand.Rand, cat zoneCatalog, z rawschema.Zone, span consts.Chunk) string {
	zc := cat[z]
	i, ok := consts.Pick(rng, zc, func(f facilityWeight) float64 {
		if span < f.minSpan {
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
	if r.layout == rawschema.DistrictLayoutCampus {
		fence = campusFence(r, l, g.chunkW, g.chunkH)
	}
	return renderUrbanChunk(world, g, ChunkSeed2D(r.seed, l.X, l.Y), r.span(), lot, l.Sub(lot.anchor), urbanDangerAt(world, c), fence)
}

// campusFence は研究所の敷地の中のチャンク l に置く塀のマスを、チャンク内の座標で返す。塀は敷地の外周を巡り、
// 真ん中の列の北と南の端に、構内の縦の街路の幅だけ門を開ける。東と南の辺には街路が無いので、建物の区画の外側の
// 1列に置く
func campusFence(r districtRegion, l consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile) []consts.Coord[consts.Tile] {
	gate := l.X == r.w/2
	var out []consts.Coord[consts.Tile]
	add := func(x, y consts.Tile) {
		if gate && ((y == 0 && l.Y == 0) || (y == chunkH-1 && l.Y == r.h-1)) && x < urbanStreetW {
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

// renderUrbanChunk は1チャンクに街路と施設の絵と塀を置き、規模に応じた敵を湧かせる。塀はチャンク内の座標で受ける。
// 部屋の絵を持つ施設は区画の中の座標 l の部屋と帯を組み、それ以外は1棟の絵を描く
func renderUrbanChunk(world w.World, g chunkGeom, seed uint64, size consts.Chunk, lot buildingLot, l consts.Coord[consts.Chunk], danger consts.Danger, fence []consts.Coord[consts.Tile]) error {
	// ストリーム識別子 0x2 は建物幾何と敵配置。施設抽選の 0x1、施設の絵の 0x3 と分けて相互干渉を避ける
	rng := rand.New(rand.NewPCG(seed, 0x2))
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return fmt.Errorf("failed to load facility templates: %w", err)
	}
	var cells [][]maptemplate.MapCell
	var indoor func(x, y int) bool
	origin := urbanStreetW
	if loader.HasRooms(lot.facility) {
		cells, indoor, err = roomChunkCells(loader, lot, l, g.chunkW, g.chunkH)
		if err != nil {
			return fmt.Errorf("failed to plan facility %q room: %w", lot.facility, err)
		}
		origin = 0
	} else {
		if err := drawUrbanStreets(world, g); err != nil {
			return err
		}
		cells, err = planFacility(loader, lot.facility, rand.New(rand.NewPCG(seed, 0x3)).Uint64(), rng)
		if err != nil {
			return fmt.Errorf("failed to plan facility %q: %w", lot.facility, err)
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
			return fmt.Errorf("failed to place campus fence at %d,%d: %w", pos.X, pos.Y, err)
		}
	}
	return spawnUrbanEnemies(world, g, rng, size, fac, danger, isWall, occupied)
}

// drawUrbanStreets はチャンクの北辺と西辺に街路を描く。建物区画のタイルは建物の生成が描くので触らない
func drawUrbanStreets(world w.World, g chunkGeom) error {
	tiles := g.tiles.get()
	for ly := range g.chunkH {
		for lx := range g.chunkW {
			if lx >= urbanStreetW && ly >= urbanStreetW {
				continue
			}
			if _, err := replaceTile(world, tiles, consts.Coord[consts.Tile]{X: g.offsetX + lx, Y: g.offsetY + ly}, consts.TileNameFloor); err != nil {
				return fmt.Errorf("failed to place urban street (x=%d, y=%d): %w", g.offsetX+lx, g.offsetY+ly, err)
			}
		}
	}
	return nil
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
