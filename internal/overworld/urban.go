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
	"github.com/mlange-42/ark/ecs"
)

// 市街地は建物チャンクの2次元格子。1チャンク = 1建物とし、街路で区切る。
// 市街地はアンカーから東と南へ w×h
// チャンク広がり、各チャンクは自分の建物を (citySeed, チャンクのローカル座標) から独立に
// 決める。全体一括導出や断片クリップは要らず、各チャンクが自己完結する。
// 街路は各チャンクの北辺・西辺に敷き、隣接チャンクと連続して格子状の街並みになる。
const (
	urbanMaxSpan consts.Chunk = 3 // 市街地の一辺の最大チャンク数

	cityStreetW    consts.Tile = 4 // チャンクの北辺・西辺の街路の幅。2車線+歩道ぶん
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

// urbanFeature は市街地の feature 実装。
type urbanFeature struct{}

// urbanRegionOf は c を含む市街地のアンカーと大きさを返す。市街地はアンカーから東と南へ
// w×h チャンク広がる。該当しなければ ok=false。走査窓には当選アンカーが複数入りうるので
// 早期に false を返さず探索を続ける。市街地どうしは urbanPlacement の Separation で重ならない
// ため c を覆うアンカーは高々1つで、最初に見つかったものを返せば一意に定まる。
func urbanRegionOf(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) (anchor consts.Coord[consts.Chunk], w, h consts.Chunk, ok bool) {
	for dy := range urbanMaxSpan {
		for dx := range urbanMaxSpan {
			a := c.Sub(consts.Coord[consts.Chunk]{X: dx, Y: dy})
			if !urbanPlacement.At(runSeed, a, rows) {
				continue
			}
			cw, ch := urbanSizeOf(ChunkSeed2D(runSeed^urbanSalt, a.X, a.Y))
			if dx < cw && dy < ch {
				return a, cw, ch, true
			}
		}
	}
	return consts.Coord[consts.Chunk]{}, 0, 0, false
}

// facilityKind は建物の施設種別。規模で gate した重み付き抽選で決まり、内装の prop の差になる。
// v1 は語彙と内装だけで、施設固有の戦利品はアイテム設計が固まってから続ける。
// 実体は文字列。%v やログで数値でなく種別名が出て、デバッグで読みやすい。
type facilityKind string

const (
	facilityHouse   facilityKind = "house"   // 住宅
	facilityStore   facilityKind = "store"   // 商店
	facilityOffice  facilityKind = "office"  // 事務所
	facilityDepot   facilityKind = "depot"   // 倉庫
	facilityAntique facilityKind = "antique" // 骨董品店
	facilityClinic  facilityKind = "clinic"  // 診療所
	facilityLab     facilityKind = "lab"     // 研究施設
)

// facilityWeight は施設の抽選重みと規模 gate。minSpan は市街地の一辺がこの値以上のときだけ
// 抽選対象になる。規模で絞る gate で、大きな市街地でだけ専門施設が混ざる。
type facilityWeight struct {
	kind    facilityKind
	weight  int
	minSpan consts.Chunk
}

// zone は市街地内の地区。中心からの位置で決まり、地区ごとに施設抽選の重みを変える。
// per-chunk 独立の抽選では隣接同種率がランダムと変わらずごま塩になるため、地区で重みを
// 揃えて空間相関を作り「地区」を生む。現代日本の市街地をイメージした語彙にする。
type zone uint8

const (
	zoneDowntown    zone = iota // 都心。商業と専門施設。最大規模の市街地の中心にだけ現れる
	zoneResidential             // 住宅地。住宅が中心
	zoneIndustrial              // 産業区。倉庫が中心
)

// zoneCatalog は地区ごとの施設抽選重み。地区で重みが揃うので同じ地区の隣接チャンクは同種へ
// 寄り、地区が生まれる。都心は必ず span=3 で現れるので専門施設の骨董品店・診療所・研究施設を
// 含められる。各地区とも span=2 の入口を持つので、規模 gate で候補が空になり抽選が壊れることはない。
var zoneCatalog = map[zone][]facilityWeight{
	zoneDowntown: {
		{facilityStore, 25, 2},
		{facilityOffice, 20, 2},
		{facilityAntique, 20, 3},
		{facilityClinic, 20, 3},
		{facilityLab, 15, 3},
	},
	zoneResidential: {
		{facilityHouse, 65, 2},
		{facilityStore, 25, 2},
		{facilityClinic, 10, 3},
	},
	zoneIndustrial: {
		{facilityDepot, 65, 2},
		{facilityOffice, 25, 2},
		{facilityHouse, 10, 2},
	},
}

// industrialCityBit は市街地の性格を住宅地寄りか産業区寄りかに振る citySeed のビット。
// 規模抽選が使う下位ビット urbanSizeOf の >>8・>>16 と衝突しない上位ビットを使う。
const industrialCityBit = uint64(1) << 32

// zoneOf は市街地内のローカルチャンク座標から地区を決める純関数。厳密な中心を都心、それ以外を
// 市街地の性格で住宅地か産業区にする。中心は 2 倍座標のチェビシェフ距離が 0 のマスで、
// 奇数×奇数すなわち 3×3 の市街地にだけ存在する。都心が必ず最大規模に出るので専門施設を集められる。
func zoneOf(lx, ly, cw, ch consts.Chunk, citySeed uint64) zone {
	dx := int(lx)*2 - int(cw-1)
	dy := int(ly)*2 - int(ch-1)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if max(dx, dy) == 0 {
		return zoneDowntown
	}
	if citySeed&industrialCityBit != 0 {
		return zoneIndustrial
	}
	return zoneResidential
}

// rollFacilityInZone は地区の重み表から規模 gate を通った施設を1つ重みで抽選する。
func rollFacilityInZone(rng *rand.Rand, z zone, span consts.Chunk) facilityKind {
	cat := zoneCatalog[z]
	total := 0
	for _, f := range cat {
		if span >= f.minSpan {
			total += f.weight
		}
	}
	roll := rng.IntN(total)
	for _, f := range cat {
		if span < f.minSpan {
			continue
		}
		roll -= f.weight
		if roll < 0 {
			return f.kind
		}
	}
	panic("到達しない: 抽選重みの合計と減算が不整合")
}

// cityChunkInfo は c が市街地の建物チャンクなら、その施設種別と市街地の規模を返す純関数。
// 地図と生成の両方がこれを呼び、地図の記号と実体の施設を一致させる。施設は地区の重みで
// 抽選するので、隣接チャンクが同じ地区なら同種へ寄る。
func cityChunkInfo(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) (kind facilityKind, size consts.Chunk, ok bool) {
	anchor, cw, ch, ok := urbanRegionOf(runSeed, c, rows)
	if !ok {
		return "", 0, false
	}
	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	chunkSeed := ChunkSeed2D(citySeed, c.X-anchor.X, c.Y-anchor.Y)
	size = max(cw, ch)
	z := zoneOf(c.X-anchor.X, c.Y-anchor.Y, cw, ch, citySeed)
	// 施設抽選は建物幾何と別の乱数ストリームにして、片方を変えても他方が動かないようにする。
	// ストリーム識別子 0x1 は施設抽選、0x2 は建物幾何と敵配置。renderCityChunk と揃える
	frng := rand.New(rand.NewPCG(chunkSeed, 0x1))
	return rollFacilityInZone(frng, z, size), size, true
}

// place は c が市街地の建物チャンクなら自分の建物を1棟描く。各チャンクは自己完結するので
// 生成順に依存しない。
func (urbanFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
	anchor, _, _, ok := urbanRegionOf(runSeed, c, rows)
	if !ok {
		return nil
	}

	// 施設種別は地図(ChunkPlace)が使う。ここでは規模だけ敵配置に使う
	_, size, _ := cityChunkInfo(runSeed, c, rows)
	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	chunkSeed := ChunkSeed2D(citySeed, c.X-anchor.X, c.Y-anchor.Y)
	return renderCityChunk(world, g, chunkSeed, size)
}

// renderCityChunk は1チャンクに建物1棟を描き、規模に応じた敵を湧かせる。
func renderCityChunk(world w.World, g chunkGeom, seed uint64, size consts.Chunk) error {
	// ストリーム識別子 0x2 は建物幾何と敵配置。施設抽選の 0x1 と分けて相互干渉を避ける
	rng := rand.New(rand.NewPCG(seed, 0x2))
	isWall, err := drawCityBuilding(world, g, rng)
	if err != nil {
		return err
	}
	return spawnCityEnemies(world, g, rng, size, isWall)
}

// drawCityBuilding は北辺・西辺の街路と、敷地をほぼ埋める建物1棟の殻を描く。建物は外周が壁・
// 内側が床で、道路に面した見える扉を持つ。街路は隣接チャンクと連続して格子になる。内装は持たない。
// 壁判定の関数を返し、敵配置が壁マスを避けるのに使う。
func drawCityBuilding(world w.World, g chunkGeom, rng *rand.Rand) (func(lx, ly consts.Tile) bool, error) {
	tiles := g.tiles.get()

	// 建物の大きさと位置。北辺・西辺の街路を避け、敷地内で余白を残して前庭や隙間を作る。
	// 建物は最小 3×3 を保証する。扉オフセット IntN(bw-2) と敷地内配置 IntN(spanX-bw+1) が
	// 破綻しない下限で、市街地チャンクは chunkW,chunkH >= cityStreetW+3 を前提にする
	spanX := g.chunkW - cityStreetW
	spanY := g.chunkH - cityStreetW
	bw := max(3, spanX-consts.Tile(rng.IntN(int(cityMaxSetback)+1)))
	bh := max(3, spanY-consts.Tile(rng.IntN(int(cityMaxSetback)+1)))
	bx := cityStreetW + consts.Tile(rng.IntN(int(spanX-bw)+1))
	by := cityStreetW + consts.Tile(rng.IntN(int(spanY-bh)+1))

	// 街路が北・西にあるので扉は道路に面する北壁か西壁に開ける。向きは壁の走る方向で決め、
	// 東西に走る北壁の切れ目は Vertical、南北に走る西壁は Horizontal。door_planner と同じ規約
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
			if err := replaceTile(world, tiles, consts.Coord[consts.Tile]{X: g.offsetX + lx, Y: g.offsetY + ly}, name); err != nil {
				return nil, fmt.Errorf("市街地の配置に失敗 (x=%d, y=%d): %w", g.offsetX+lx, g.offsetY+ly, err)
			}
		}
	}
	// 開口に道路へ面した見える扉を置く。1マスの床の切れ目だけでは入口と分からないため明示する
	if _, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: g.offsetX + doorX, Y: g.offsetY + doorY}, doorOrient); err != nil {
		return nil, fmt.Errorf("市街地の扉配置に失敗: %w", err)
	}
	return isWall, nil
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
func replaceTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile], tileName string) error {
	g := gc.GridElement{Coord: pos}
	if e, ok := tiles[g]; ok && world.ECS.Alive(e) {
		world.ECS.RemoveEntity(e)
	}
	zero := 0
	e, err := lifecycle.SpawnTile(world, tileName, pos.X, pos.Y, &zero)
	if err != nil {
		return err
	}
	// 置換後の実体を索引へ書き戻す。索引を地物間で共有するため、後続の地物が同じ座標を
	// 正しく再置換でき、旧タイルの二重残留を防ぐ
	tiles[g] = e
	return nil
}
