package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
)

// 市街地は建物チャンクの2次元格子。CDDA の街が OMT ごとに1つの建物で埋まり街路で
// 区切られるのを翻案し、1チャンク = 1建物にする。市街地はアンカーから東と南へ w×h
// チャンク広がり、各チャンクは自分の建物を (citySeed, チャンクのローカル座標) から独立に
// 決める。全体一括導出や断片クリップは要らず、各チャンクが自己完結する。
// 街路は各チャンクの北辺・西辺に敷き、隣接チャンクと連続して格子状の街並みになる。
const (
	urbanSalt                 = 0x0b17
	urbanMaxSpan consts.Chunk = 3 // 市街地の一辺の最大チャンク数

	cityStreetW    consts.Tile = 4 // チャンクの北辺・西辺の街路の幅。CDDA の2車線+歩道に相当
	cityMaxSetback consts.Tile = 3 // 建物を敷地内で縮めてよい最大量。前庭や隙間を作る

	// urbanEnemyTable は市街地の敵抽選に使う敵テーブル名。市街地の規模を深度とみなして引く
	urbanEnemyTable = "廃墟"
)

// urbanSizeOf は市街地の縦横のチャンク数を citySeed から決定的に選ぶ。各辺 2..urbanMaxSpan。
// 規模が大きいほど敵も戦利品も多く、レアな施設が混ざる。リスクとリターンが規模に比例する。
func urbanSizeOf(citySeed uint64) (w, h consts.Chunk) {
	span := uint64(urbanMaxSpan - 1)
	w = 2 + consts.Chunk((citySeed>>8)%span)
	h = 2 + consts.Chunk((citySeed>>16)%span)
	return w, h
}

// urbanPlacement は市街地アンカーのリージョン配置。小集落より疎に置く。
// 最大辺 urbanMaxSpan より広い間隔にして、隣り合う市街地が重ならないようにする。
var urbanPlacement = Placement{Spacing: 6, Separation: 2, Salt: urbanSalt}

// urbanRuinFeature は市街地の feature 実装。
type urbanRuinFeature struct{}

// urbanRegionOf は c を含む市街地のアンカーと大きさを返す。市街地はアンカーから東と南へ
// w×h チャンク広がる。該当しなければ ok=false。走査窓に当選アンカーが複数入りうるので、
// 早期に false を返さず、c を覆うアンカーを探し続けて最も近いものを採る。
func urbanRegionOf(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) (anchor worldstream.ChunkCoord, w, h consts.Chunk, ok bool) {
	for dy := range urbanMaxSpan {
		for dx := range urbanMaxSpan {
			a := worldstream.ChunkCoord{X: c.X - dx, Y: c.Y - dy}
			if !urbanPlacement.At(runSeed, a, rows) {
				continue
			}
			cw, ch := urbanSizeOf(ChunkSeed2D(runSeed^urbanSalt, a.X, a.Y))
			if dx < cw && dy < ch {
				return a, cw, ch, true
			}
		}
	}
	return worldstream.ChunkCoord{}, 0, 0, false
}

// facilityKind は建物の施設種別。規模で gate した重み付き抽選で決まり、内装の prop の差になる。
// v1 は語彙と内装だけで、施設固有の戦利品はアイテム設計が固まってから続ける。
type facilityKind uint8

const (
	facilityHouse   facilityKind = iota // 住宅
	facilityStore                       // 商店
	facilityOffice                      // 事務所
	facilityDepot                       // 倉庫
	facilityAntique                     // 骨董品店
	facilityClinic                      // 診療所
	facilityLab                         // 研究施設
)

// facilityCatalog は施設の抽選重みと規模 gate と内装。minSpan は市街地の一辺がこの値以上の
// ときだけ抽選対象になる。CDDA の city_size gate の翻案で、大きな市街地でだけ専門施設と
// レアの研究施設が混ざる。現代日本の市街地をイメージした語彙にする。
var facilityCatalog = []struct {
	kind    facilityKind
	weight  int
	minSpan consts.Chunk
}{
	{facilityHouse, 40, 2},
	{facilityStore, 20, 2},
	{facilityOffice, 15, 2},
	{facilityDepot, 15, 2},
	{facilityAntique, 8, 3},
	{facilityClinic, 8, 3},
	{facilityLab, 3, 3},
}

// placeAnchor は家具を建物内のどこに置くかの意味。ランダム散布でなく「役割と位置の意味」で
// 配置する。これが「店に見える／住居に見える」を生む。
type placeAnchor uint8

const (
	anchorAlongWall placeAnchor = iota // 壁の内側に沿って
	anchorNearDoor                     // 入口の内側の脇。レジ・受付
	anchorAisle                        // 通路状に平行な棚の列
	anchorCenter                       // 部屋の中央。机・卓
	anchorCorner                       // 隅。ベッド・ロッカー
)

// furnishSpec は家具1種の配置規則。fill=true はそのアンカーの空きを埋め、false は1個置く。
type furnishSpec struct {
	name   string
	anchor placeAnchor
	fill   bool
}

// roomFurnish は施設の部屋の役割ごとの内装規則。front は入口を含む表の部屋、back は奥の部屋。
// 部屋分割で表と奥に別の家具を置くので、店舗フロアと在庫置き場、居間と寝室のように読める。
type roomFurnish struct {
	front []furnishSpec // 表の部屋。店舗フロア・居間・受付
	back  []furnishSpec // 奥の部屋。在庫置き場・寝室・機械室
}

// facilityFurnish は施設種別ごとの内装配置規則。完成マップでなく規則を authoring する。
// 「レジは入口脇」「棚は通路状」「寝室は奥」という少数の規則が、無限の建物を店や住居らしく見せる。
var facilityFurnish = map[facilityKind]roomFurnish{
	facilityHouse: {
		front: []furnishSpec{{"sofa", anchorAlongWall, false}, {"table", anchorCenter, false}, {"chair", anchorCenter, false}, {"tall_lamp", anchorCorner, false}},
		back:  []furnishSpec{{"bed", anchorCorner, false}, {"closet", anchorCorner, false}, {"drawer_chest", anchorAlongWall, false}},
	},
	facilityStore: {
		front: []furnishSpec{{"register", anchorNearDoor, false}, {"goods_shelf", anchorAisle, true}},
		back:  []furnishSpec{{"crate", anchorAlongWall, true}, {"barrel", anchorCorner, false}},
	},
	facilityOffice: {
		front: []furnishSpec{{"desk", anchorCenter, false}, {"desktop_pc", anchorCenter, false}, {"chair", anchorCenter, false}},
		back:  []furnishSpec{{"bookshelf", anchorAlongWall, true}, {"electric_locker", anchorCorner, false}},
	},
	facilityDepot: {
		front: []furnishSpec{{"iron_shelf", anchorAisle, true}},
		back:  []furnishSpec{{"crate", anchorAlongWall, true}, {"barrel", anchorCorner, false}},
	},
	facilityAntique: {
		front: []furnishSpec{{"register", anchorNearDoor, false}, {"artistic_shelf", anchorAlongWall, true}, {"book_showcase", anchorAlongWall, false}},
		back:  []furnishSpec{{"黒い花瓶", anchorCorner, false}, {"old_lamp", anchorCorner, false}},
	},
	facilityClinic: {
		front: []furnishSpec{{"bed", anchorAlongWall, true}, {"sink", anchorCorner, false}},
		back:  []furnishSpec{{"ロッカー", anchorCorner, false}, {"electric_locker", anchorCorner, false}},
	},
	facilityLab: {
		front: []furnishSpec{{"gauge_machine", anchorAlongWall, true}, {"desktop_pc", anchorCenter, false}},
		back:  []furnishSpec{{"generator_green", anchorCorner, false}, {"electric_locker", anchorCorner, false}},
	},
}

// rollFacility は規模 gate を通った施設を重みで1つ抽選し、facilityCatalog の添字を返す。
func rollFacility(rng *rand.Rand, span consts.Chunk) int {
	total := 0
	for _, f := range facilityCatalog {
		if span >= f.minSpan {
			total += f.weight
		}
	}
	roll := rng.IntN(total)
	for i, f := range facilityCatalog {
		if span < f.minSpan {
			continue
		}
		roll -= f.weight
		if roll < 0 {
			return i
		}
	}
	return 0
}

// cityChunkInfo は c が市街地の建物チャンクなら、その施設種別と市街地の規模を返す純関数。
// 地図と生成の両方がこれを呼び、地図の記号と実体の施設を一致させる。
func cityChunkInfo(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) (facility int, size consts.Chunk, ok bool) {
	anchor, cw, ch, ok := urbanRegionOf(runSeed, c, rows)
	if !ok {
		return 0, 0, false
	}
	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	chunkSeed := ChunkSeed2D(citySeed, c.X-anchor.X, c.Y-anchor.Y)
	size = max(cw, ch)
	// 施設抽選は建物幾何と別の乱数ストリームにして、片方を変えても他方が動かないようにする
	frng := rand.New(rand.NewPCG(chunkSeed, 0x1))
	return rollFacility(frng, size), size, true
}

// place は c が市街地の建物チャンクなら自分の建物を1棟描く。市街地が開始チャンクを含むなら
// 丸ごとスキップし、新規ゲームの開始点を安全に保つ。各チャンクは自己完結するので生成順に
// 依存しない。
func (urbanRuinFeature) place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	anchor, cw, ch, ok := urbanRegionOf(runSeed, c, rows)
	if !ok {
		return nil
	}
	for dy := range ch {
		for dx := range cw {
			if (worldstream.ChunkCoord{X: anchor.X + dx, Y: anchor.Y + dy}) == start {
				return nil
			}
		}
	}

	facility, size, _ := cityChunkInfo(runSeed, c, rows)
	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	chunkSeed := ChunkSeed2D(citySeed, c.X-anchor.X, c.Y-anchor.Y)
	return renderCityChunk(world, g, chunkSeed, facility, size)
}

// renderCityChunk は1チャンクに建物1棟を描き、規模に応じた敵を湧かせる。
func renderCityChunk(world w.World, g chunkGeom, seed uint64, facility int, size consts.Chunk) error {
	rng := rand.New(rand.NewPCG(seed, 0x2))
	isWall, err := drawCityBuilding(world, g, rng, facility)
	if err != nil {
		return err
	}
	return spawnCityEnemies(world, g, rng, size, isWall)
}

// drawCityBuilding は北辺・西辺の街路と、敷地をほぼ埋める建物1棟を描く。建物は外周が壁・
// 内側が床で、道路に面した見える扉を持ち、屋内は役割ベースで内装する。街路は隣接チャンクと
// 連続して格子になる。壁判定の関数を返し、敵配置が壁マスを避けるのに使う。敵は含めないので、
// 単独の建物サンプル生成からも呼べる。
func drawCityBuilding(world w.World, g chunkGeom, rng *rand.Rand, facility int) (func(lx, ly consts.Tile) bool, error) {
	tiles := tileEntitiesInRange(world, g.offsetX, g.offsetX+g.chunkW)

	// 建物の大きさと位置。北辺・西辺の街路を避け、敷地内で余白を残して前庭や隙間を作る
	spanX := g.chunkW - cityStreetW
	spanY := g.chunkH - cityStreetW
	bw := spanX - consts.Tile(rng.IntN(int(cityMaxSetback)+1))
	bh := spanY - consts.Tile(rng.IntN(int(cityMaxSetback)+1))
	bx := cityStreetW + consts.Tile(rng.IntN(int(spanX-bw)+1))
	by := cityStreetW + consts.Tile(rng.IntN(int(spanY-bh)+1))

	// 街路は各チャンクの北辺・西辺にあるので、扉は道路に面する北壁か西壁のどちらかに開ける。
	// 向きは壁の走る方向で決める。東西に走る北壁の切れ目は Vertical、南北に走る西壁は Horizontal。
	// door_planner の doorOrientation と同じ規約に合わせる
	doorX, doorY := bx+1+consts.Tile(rng.IntN(int(bw-2))), by
	doorOrient := gc.DoorOrientationVertical
	if rng.IntN(2) == 0 {
		doorX, doorY = bx, by+1+consts.Tile(rng.IntN(int(bh-2)))
		doorOrient = gc.DoorOrientationHorizontal
	}

	// 屋内を部屋に分割し、仕切り壁とドアを得る。大部屋を廊下と部屋に割る文法(レバーB)
	interior := rrect{x0: bx + 1, y0: by + 1, x1: bx + bw - 2, y1: by + bh - 2}
	rooms, walls := subdivideBuilding(rng, interior)
	internalWall := map[consts.Coord[consts.Tile]]bool{}
	for _, seg := range walls {
		for x := seg.x0; x <= seg.x1; x++ {
			for y := seg.y0; y <= seg.y1; y++ {
				if x == seg.doorX && y == seg.doorY {
					continue // 部屋をつなぐドアの開口
				}
				internalWall[consts.Coord[consts.Tile]{X: x, Y: y}] = true
			}
		}
	}

	inBuilding := func(lx, ly consts.Tile) bool {
		return lx >= bx && lx < bx+bw && ly >= by && ly < by+bh
	}
	isWall := func(lx, ly consts.Tile) bool {
		if !inBuilding(lx, ly) {
			return false
		}
		perimeter := lx == bx || lx == bx+bw-1 || ly == by || ly == by+bh-1
		if perimeter {
			return lx != doorX || ly != doorY
		}
		return internalWall[consts.Coord[consts.Tile]{X: lx, Y: ly}]
	}
	for ly := range g.chunkH {
		for lx := range g.chunkW {
			name := ""
			switch {
			case lx < cityStreetW || ly < cityStreetW:
				name = consts.TileNameFloor // 街路
			case isWall(lx, ly):
				name = consts.TileNameDWall
			case inBuilding(lx, ly):
				name = consts.TileNameFloor // 屋内・出入口・部屋をつなぐドア
			}
			if name == "" {
				continue // 前庭・空き地は土のまま残す
			}
			if err := replaceTile(world, tiles, g.offsetX+lx, g.offsetY+ly, name); err != nil {
				return nil, fmt.Errorf("市街地の配置に失敗 (x=%d, y=%d): %w", g.offsetX+lx, g.offsetY+ly, err)
			}
		}
	}

	// 入口の内側マス。この部屋を「表」とし、他は「奥」として別の家具を置く
	dinX, dinY := doorX, doorY+1
	if doorX == bx {
		dinX, dinY = doorX+1, doorY
	}
	kind := facilityCatalog[facility].kind
	for _, room := range rooms {
		isEntrance := room.contains(dinX, dinY)
		specs := facilityFurnish[kind].back
		if isEntrance {
			specs = facilityFurnish[kind].front
		}
		if err := furnishRoom(world, g, rng, specs, room, dinX, dinY, isEntrance); err != nil {
			return nil, err
		}
	}
	// 開口に道路へ面した見える扉を置く。1マスの床の切れ目だけでは入口と分からないため明示する
	if _, err := lifecycle.SpawnDoor(world, g.offsetX+doorX, g.offsetY+doorY, doorOrient); err != nil {
		return nil, fmt.Errorf("市街地の扉配置に失敗: %w", err)
	}
	return isWall, nil
}

// cityMinRoom は部屋の最小の一辺。これを下回る幅には割らない。
const cityMinRoom consts.Tile = 3

// cityRoomDepth は部屋分割の再帰段数。1建物で最大 2^depth 部屋になる。
const cityRoomDepth = 2

// furnishRoom は1部屋の床範囲 [ix0,ix1]×[iy0,iy1] を役割ベースで内装する。ランダム散布でなく、
// 家具ごとに定めた位置の意味(壁沿い・入口脇・通路・中央・隅)に従って置く。isEntrance が真の
// 部屋だけレジ・受付を入口脇に置く。入口の内側1マスは通行のため空ける。すべて決定的な手続き。
func furnishRoom(world w.World, g chunkGeom, rng *rand.Rand, specs []furnishSpec, room rrect, dinX, dinY consts.Tile, isEntrance bool) error {
	ix0, iy0, ix1, iy1 := room.x0, room.y0, room.x1, room.y1
	if ix1 < ix0 || iy1 < iy0 {
		return nil
	}
	occupied := map[consts.Coord[consts.Tile]]bool{}
	if isEntrance {
		occupied[consts.Coord[consts.Tile]{X: dinX, Y: dinY}] = true // 入口の導線を空ける
	}

	place := func(name string, x, y consts.Tile) (bool, error) {
		if x < ix0 || x > ix1 || y < iy0 || y > iy1 {
			return false, nil
		}
		p := consts.Coord[consts.Tile]{X: x, Y: y}
		if occupied[p] {
			return false, nil
		}
		occupied[p] = true
		if _, err := lifecycle.SpawnProp(world, name, g.offsetX+x, g.offsetY+y); err != nil {
			return false, fmt.Errorf("市街地の内装配置に失敗 (%s): %w", name, err)
		}
		return true, nil
	}

	for _, s := range specs {
		if s.anchor == anchorNearDoor && !isEntrance {
			continue // レジ・受付は入口のある表の部屋だけ
		}
		cells := anchorCells(s.anchor, ix0, iy0, ix1, iy1, dinX, dinY, rng)
		for _, c := range cells {
			ok, err := place(s.name, c.X, c.Y)
			if err != nil {
				return err
			}
			if ok && !s.fill {
				break // 1個置いたら次の家具へ
			}
		}
	}
	return nil
}

// anchorCells はアンカーの意味に対応する屋内の候補マスを、置く順に返す。
// 内寸は [ix0,ix1]×[iy0,iy1] の閉区間。din は入口内側で、通路として避ける。
func anchorCells(a placeAnchor, ix0, iy0, ix1, iy1, dinX, dinY consts.Tile, rng *rand.Rand) []consts.Coord[consts.Tile] {
	pt := func(x, y consts.Tile) consts.Coord[consts.Tile] { return consts.Coord[consts.Tile]{X: x, Y: y} }
	switch a {
	case anchorCorner:
		// 4隅。順序を seed で回して建物ごとに散らす
		cs := []consts.Coord[consts.Tile]{pt(ix0, iy0), pt(ix1, iy0), pt(ix0, iy1), pt(ix1, iy1)}
		rng.Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
		return cs
	case anchorNearDoor:
		// 入口内側の左右。レジ・受付を入口脇に置く
		return []consts.Coord[consts.Tile]{pt(dinX-1, dinY), pt(dinX+1, dinY), pt(dinX, dinY+1)}
	case anchorCenter:
		var cs []consts.Coord[consts.Tile]
		for y := iy0 + 1; y <= iy1-1; y++ {
			for x := ix0 + 1; x <= ix1-1; x++ {
				cs = append(cs, pt(x, y))
			}
		}
		return cs
	case anchorAisle:
		// 1列おきに棚の列を作り、間を通路として空ける。壁から1マス空けて圧迫を避ける
		var cs []consts.Coord[consts.Tile]
		for x := ix0 + 1; x <= ix1-1; x += 2 {
			for y := iy0; y <= iy1; y++ {
				cs = append(cs, pt(x, y))
			}
		}
		return cs
	default: // anchorAlongWall
		// 内寸の外周リングを一周。壁沿いに家具を並べる
		var cs []consts.Coord[consts.Tile]
		for x := ix0; x <= ix1; x++ {
			cs = append(cs, pt(x, iy0))
		}
		for x := ix0; x <= ix1; x++ {
			cs = append(cs, pt(x, iy1))
		}
		for y := iy0 + 1; y <= iy1-1; y++ {
			cs = append(cs, pt(ix0, y), pt(ix1, y))
		}
		return cs
	}
}

// spawnCityEnemies はチャンクに敵を数体湧かせる。数は市街地の規模に比例し、種類は敵テーブルから
// 規模を深度とみなして重み抽選する。壁マスに埋まる位置は避ける。
func spawnCityEnemies(world w.World, g chunkGeom, rng *rand.Rand, size consts.Chunk, isWall func(lx, ly consts.Tile) bool) error {
	enemyTable, err := raw.GetEnemyTable(world.Resources.RawMaster, urbanEnemyTable)
	if err != nil {
		return fmt.Errorf("市街地の敵テーブル取得に失敗: %w", err)
	}
	count := 1 + rng.IntN(int(size))
	for range count {
		lx := consts.Tile(rng.IntN(int(g.chunkW)))
		ly := consts.Tile(rng.IntN(int(g.chunkH)))
		enemyName, err := raw.SelectEnemyByWeight(enemyTable, rng, int(size))
		if err != nil {
			return fmt.Errorf("市街地の敵抽選に失敗: %w", err)
		}
		if isWall(lx, ly) {
			continue // 抽選は消費済みなので決定性は保たれる
		}
		pos := consts.Coord[consts.Tile]{X: g.offsetX + lx, Y: g.offsetY + ly}
		if _, err := lifecycle.SpawnEnemy(world, pos, enemyName); err != nil {
			return fmt.Errorf("市街地の敵配置に失敗: %w", err)
		}
	}
	return nil
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
func replaceTile(world w.World, tiles map[gc.GridElement]ecs.Entity, x, y consts.Tile, tileName string) error {
	g := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	if e, ok := tiles[g]; ok && world.ECS.Alive(e) {
		world.ECS.RemoveEntity(e)
		delete(tiles, g)
	}
	zero := 0
	if _, err := lifecycle.SpawnTile(world, tileName, x, y, &zero); err != nil {
		return err
	}
	return nil
}
