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
	props   []string
}{
	{facilityHouse, 40, 2, []string{"bed", "closet", "table", "refrigerator"}},
	{facilityStore, 20, 2, []string{"register", "goods_shelf", "goods_shelf", "refrigerator"}},
	{facilityOffice, 15, 2, []string{"desk", "desktop_pc", "chair", "bookshelf"}},
	{facilityDepot, 15, 2, []string{"crate", "crate", "barrel", "iron_shelf"}},
	{facilityAntique, 8, 3, []string{"artistic_shelf", "book_showcase", "黒い花瓶", "old_lamp"}},
	{facilityClinic, 8, 3, []string{"bed", "bed", "sink", "ロッカー"}},
	{facilityLab, 3, 3, []string{"gauge_machine", "generator_green", "electric_locker", "desktop_pc"}},
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

// renderCityChunk は1チャンクに、北辺・西辺の街路と、敷地をほぼ埋める建物を1棟描く。
// 建物は外周が壁・内側が床で、南辺に見える扉を持つ。街路は隣接チャンクと連続して格子になる。
func renderCityChunk(world w.World, g chunkGeom, seed uint64, facility int, size consts.Chunk) error {
	rng := rand.New(rand.NewPCG(seed, 0x2))
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

	inBuilding := func(lx, ly consts.Tile) bool {
		return lx >= bx && lx < bx+bw && ly >= by && ly < by+bh
	}
	isWall := func(lx, ly consts.Tile) bool {
		if !inBuilding(lx, ly) {
			return false
		}
		perimeter := lx == bx || lx == bx+bw-1 || ly == by || ly == by+bh-1
		return perimeter && (lx != doorX || ly != doorY)
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
				name = consts.TileNameFloor // 屋内・出入口
			}
			if name == "" {
				continue // 前庭・空き地は土のまま残す
			}
			if err := replaceTile(world, tiles, g.offsetX+lx, g.offsetY+ly, name); err != nil {
				return fmt.Errorf("市街地の配置に失敗 (x=%d, y=%d): %w", g.offsetX+lx, g.offsetY+ly, err)
			}
		}
	}

	if err := spawnBuildingProps(world, g, rng, facility, bx, by, bw, bh, doorX, doorY); err != nil {
		return err
	}
	// 開口に道路へ面した見える扉を置く。1マスの床の切れ目だけでは入口と分からないため明示する
	if _, err := lifecycle.SpawnDoor(world, g.offsetX+doorX, g.offsetY+doorY, doorOrient); err != nil {
		return fmt.Errorf("市街地の扉配置に失敗: %w", err)
	}
	return spawnCityEnemies(world, g, rng, size, isWall)
}

// spawnBuildingProps は建物の屋内へ施設種別の内装 prop をまばらに置く。広い屋内を埋めるため
// 内装候補を循環させ、空きマスに面積比例で配置する。出入口の直上は導線として空ける。
func spawnBuildingProps(world w.World, g chunkGeom, rng *rand.Rand, facility int, bx, by, bw, bh, doorX, doorY consts.Tile) error {
	type cell struct{ x, y consts.Tile }
	var free []cell
	for ly := by + 1; ly < by+bh-1; ly++ {
		for lx := bx + 1; lx < bx+bw-1; lx++ {
			// 出入口の内側の1マスは通行のため空ける
			if (lx == doorX && ly == doorY+1) || (lx == doorX+1 && ly == doorY) {
				continue
			}
			free = append(free, cell{lx, ly})
		}
	}
	names := facilityCatalog[facility].props
	count := min(len(free)/8, 16)
	for i := range count {
		if len(free) == 0 {
			break
		}
		pick := rng.IntN(len(free))
		p := free[pick]
		free[pick] = free[len(free)-1]
		free = free[:len(free)-1]
		if _, err := lifecycle.SpawnProp(world, names[i%len(names)], g.offsetX+p.x, g.offsetY+p.y); err != nil {
			return fmt.Errorf("市街地の内装配置に失敗 (%s): %w", names[i%len(names)], err)
		}
	}
	return nil
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
