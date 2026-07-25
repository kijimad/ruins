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

// 市街地は複数チャンクに及ぶ地表の危険地帯。補給の場ではなく、変質した群れが徘徊する
// 攻略対象で、舗装跡の床ブロック・施設種別を持つ街区・敵で表す。
//
// レイアウトは citySeed から市街地全体を一括導出し、各チャンクは自分の断片だけを描く。
// 断片の導出は (runSeed, アンカー座標) の純関数なので、隣接チャンクを生成せずに断片どうしが
// 一致する。帯ストリーミングで市街地の一部だけが先に生成されても矛盾しない。
const (
	urbanSalt                  = 0x0b17
	urbanMaxWidth consts.Chunk = 3 // 市街地の最大横幅。チャンク数

	// 街区の格子。CDDA の街が OMT ごとに建物で埋まり街路で区切られるのを翻案する。
	// 市街地を cityLotPitch 間隔の区画に切り、各区画をほぼ埋める大きな建物を置き、
	// 区画の西辺と北辺に幅 cityStreetW の街路を敷いて格子状の街並みにする。
	cityLotPitch   consts.Tile = 18 // 区画の間隔。建物 + 街路
	cityStreetW    consts.Tile = 4  // 街路の幅。CDDA の2車線+歩道に相当する広さにする
	cityMaxSetback consts.Tile = 4  // 区画内で建物を縮めてよい最大量。前庭や隙間を作る
	cityVacancy                = 2  // 10 区画あたり空き地にする数。単調さを崩す

	// urbanEnemyTable は市街地の敵抽選に使う敵テーブル名。市街地の規模を深度とみなして引く
	urbanEnemyTable = "廃墟"
)

// urbanWidthOf は市街地の横幅チャンク数を citySeed から決定的に選ぶ。2..urbanMaxWidth。
// 規模が大きいほど敵も多く、リスクとリターンが規模に比例する。
func urbanWidthOf(citySeed uint64) consts.Chunk {
	return 2 + consts.Chunk((citySeed>>8)%uint64(urbanMaxWidth-1))
}

// urbanPlacement は市街地アンカーのリージョン配置。小集落より疎に置く。
// チャンクは50タイルあるため、Spacing 6 で300タイルに1つの体感密度になる。
var urbanPlacement = Placement{Spacing: 6, Separation: 2, Salt: urbanSalt}

// urbanRuinFeature は市街地の feature 実装。
type urbanRuinFeature struct{}

// urbanAnchorOf は c を含む市街地のアンカーと横幅を返す。市街地はアンカーから東へ
// width チャンク続く。該当しなければ ok=false。Spacing が最大幅より大きいので、
// 走査範囲に当選アンカーは高々1つしか無い。
func urbanAnchorOf(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) (anchor worldstream.ChunkCoord, width consts.Chunk, ok bool) {
	for dx := range urbanMaxWidth {
		a := worldstream.ChunkCoord{X: c.X - dx, Y: c.Y}
		if !urbanPlacement.At(runSeed, a, rows) {
			continue
		}
		w := urbanWidthOf(ChunkSeed2D(runSeed^urbanSalt, a.X, a.Y))
		if dx < w {
			return a, w, true
		}
		return worldstream.ChunkCoord{}, 0, false
	}
	return worldstream.ChunkCoord{}, 0, false
}

// facilityKind は街区1棟の施設種別。規模で gate した重み付き抽選で決まり、内装の prop の
// 差になる。v1 は語彙と内装だけで、施設固有の戦利品はアイテム設計が固まってから続ける。
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

// facilityCatalog は施設の抽選重みと規模 gate と内装。minWidth は市街地の横幅がこの値以上の
// ときだけ抽選対象になる。CDDA の city_size gate の翻案で、大都市(幅3)でだけ専門施設と
// レアの研究施設が混ざる。現代日本の市街地をイメージした語彙にする。
var facilityCatalog = []struct {
	kind     facilityKind
	weight   int
	minWidth consts.Chunk
	props    []string
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
func rollFacility(rng *rand.Rand, width consts.Chunk) int {
	total := 0
	for _, f := range facilityCatalog {
		if width >= f.minWidth {
			total += f.weight
		}
	}
	roll := rng.IntN(total)
	for i, f := range facilityCatalog {
		if width < f.minWidth {
			continue
		}
		roll -= f.weight
		if roll < 0 {
			return i
		}
	}
	return 0
}

// cityProp は街区の内装 prop の1つ。座標は市街地ローカル。
type cityProp struct {
	name string
	x, y consts.Tile
}

// cityBuilding は市街地レイアウト上の街区1棟。座標は市街地ローカル。
type cityBuilding struct {
	x, y, w, h consts.Tile
	door       consts.Tile // 南辺の出入口の X
	facility   int         // facilityCatalog の添字
	props      []cityProp
}

// cityTile は市街地レイアウト上の1マスの種別。
type cityTile uint8

const (
	cityFloor cityTile = iota + 1 // 舗装跡・屋内の床
	cityWall                      // 廃屋の壁
)

// cityLayout は citySeed から市街地全体の街区一覧を一括導出する純関数。
// 市街地を cityLotPitch 間隔の区画格子に切り、各区画に区画をほぼ埋める建物を1棟置く。
// 一部の区画は空き地にして単調さを崩す。区画は市街地ローカル座標で切るので、どの
// チャンクから生成しても格子と建物が一致し、断片描画が矛盾しない。
func cityLayout(citySeed uint64, width consts.Chunk, cityW, cityH consts.Tile) []cityBuilding {
	rng := rand.New(rand.NewPCG(citySeed, 0))
	var buildings []cityBuilding
	for lotY := consts.Tile(0); lotY+cityLotPitch <= cityH; lotY += cityLotPitch {
		for lotX := consts.Tile(0); lotX+cityLotPitch <= cityW; lotX += cityLotPitch {
			if rng.IntN(10) < cityVacancy {
				continue // 空き地。それでも rng 列は進めないので断片間で一致する
			}
			// 区画内部の使える範囲。西辺・北辺の街路を避ける
			span := cityLotPitch - cityStreetW
			// 建物の大きさをばらつかせ、区画内で余白を残す。均一な格子の違和感を崩す。
			// 余白ぶん建物を区画内でずらすので、前庭や隣家との隙間ができる
			bw := span - consts.Tile(rng.IntN(int(cityMaxSetback)+1))
			bh := span - consts.Tile(rng.IntN(int(cityMaxSetback)+1))
			ox := consts.Tile(rng.IntN(int(span-bw) + 1))
			oy := consts.Tile(rng.IntN(int(span-bh) + 1))
			b := cityBuilding{
				x: lotX + cityStreetW + ox,
				y: lotY + cityStreetW + oy,
				w: bw,
				h: bh,
			}
			b.facility = rollFacility(rng, width)
			b.door = b.x + 1 + consts.Tile(rng.IntN(int(b.w-2))) // 南辺の出入口
			b.props = rollBuildingProps(rng, b)
			buildings = append(buildings, b)
		}
	}
	return buildings
}

// rollBuildingProps は街区の内装 prop の配置を決定的に選ぶ。施設種別の内装候補から、
// 屋内の空きマスへ順に置く。出入口の直上マスは通行のため空ける。屋内が狭ければ置ける
// ぶんだけで打ち切る。
func rollBuildingProps(rng *rand.Rand, b cityBuilding) []cityProp {
	type cell struct{ x, y consts.Tile }
	var free []cell
	for ly := b.y + 1; ly < b.y+b.h-1; ly++ {
		for lx := b.x + 1; lx < b.x+b.w-1; lx++ {
			if lx == b.door && ly == b.y+b.h-2 {
				continue // 出入口の直上
			}
			free = append(free, cell{lx, ly})
		}
	}
	names := facilityCatalog[b.facility].props
	// 広い屋内をまばらに埋める。内装候補を循環させ、面積に比例した数を置く。
	// 密度を抑えるため空きマス 8 につき 1 個ほど、上限は 16 個にする
	count := min(len(free)/8, 16)
	props := make([]cityProp, 0, count)
	for i := range count {
		if len(free) == 0 {
			break
		}
		pick := rng.IntN(len(free))
		props = append(props, cityProp{name: names[i%len(names)], x: free[pick].x, y: free[pick].y})
		free[pick] = free[len(free)-1]
		free = free[:len(free)-1]
	}
	return props
}

// cityTilesOf は街区一覧と市街地の寸法から壁・床レイアウトを導出する。
// まず区画格子の街路を舗装し、次に各建物の外周を壁・内側を床にして南辺に出入口を開ける。
// 建物は自区画の内部だけを占め互いに重ならないので、街路と建物は排他になる。
// 空き地の区画は地物を置かず、断片描画時に土のまま残る。
func cityTilesOf(buildings []cityBuilding, cityW, cityH consts.Tile) map[consts.Coord[consts.Tile]]cityTile {
	m := map[consts.Coord[consts.Tile]]cityTile{}
	// 街路の格子。各区画の西辺と北辺に幅 cityStreetW の舗装を敷く
	for y := range cityH {
		for x := range cityW {
			if x%cityLotPitch < cityStreetW || y%cityLotPitch < cityStreetW {
				m[consts.Coord[consts.Tile]{X: x, Y: y}] = cityFloor
			}
		}
	}
	// 建物。外周を壁、内側を床にし、南辺中央に出入口を開ける
	for _, b := range buildings {
		for ly := b.y; ly < b.y+b.h; ly++ {
			for lx := b.x; lx < b.x+b.w; lx++ {
				p := consts.Coord[consts.Tile]{X: lx, Y: ly}
				perimeter := lx == b.x || lx == b.x+b.w-1 || ly == b.y || ly == b.y+b.h-1
				switch {
				case perimeter && ly == b.y+b.h-1 && lx == b.door:
					m[p] = cityFloor // 出入口
				case perimeter:
					m[p] = cityWall
				default:
					m[p] = cityFloor
				}
			}
		}
	}
	return m
}

// place は c が市街地の一部なら自分の断片を描く。開始チャンクを含む市街地は丸ごと
// スキップし、新規ゲームの開始点を安全に保つ。
func (urbanRuinFeature) place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	anchor, width, ok := urbanAnchorOf(runSeed, c, rows)
	if !ok {
		return nil
	}
	for dx := range width {
		if (worldstream.ChunkCoord{X: anchor.X + dx, Y: anchor.Y}) == start {
			return nil
		}
	}

	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	fragIdx := c.X - anchor.X
	fragOrigin := fragIdx.Tiles(g.chunkW) // 市街地ローカルでの自断片の西端 X

	// 断片範囲に重なる壁・床を置換する。オートタイル添字は置換後にチャンク全域の
	// 再計算が実状態から揃えるため、ここでは仮の 0 で置いてよい
	tiles := tileEntitiesInRange(world, g.offsetX, g.offsetX+g.chunkW)
	cityW := width.Tiles(g.chunkW)
	buildings := cityLayout(citySeed, width, cityW, g.chunkH)
	layout := cityTilesOf(buildings, cityW, g.chunkH)
	inFrag := func(x consts.Tile) bool { return x >= fragOrigin && x < fragOrigin+g.chunkW }
	for p, kind := range layout {
		if !inFrag(p.X) {
			continue
		}
		wx := g.offsetX + (p.X - fragOrigin)
		wy := g.offsetY + p.Y
		name := consts.TileNameFloor
		if kind == cityWall {
			name = consts.TileNameDWall
		}
		if err := replaceTile(world, tiles, wx, wy, name); err != nil {
			return fmt.Errorf("市街地の配置に失敗 (x=%d, y=%d): %w", wx, wy, err)
		}
	}

	// 内装 prop は市街地全体で一括導出済みなので、自断片に入るものだけを実体化する。
	// 街区が断片境界をまたいでも、両断片の導出結果が一致するので矛盾しない
	for _, b := range buildings {
		for _, p := range b.props {
			if !inFrag(p.x) {
				continue
			}
			wx := g.offsetX + (p.x - fragOrigin)
			wy := g.offsetY + p.y
			if _, err := lifecycle.SpawnProp(world, p.name, wx, wy); err != nil {
				return fmt.Errorf("市街地の内装配置に失敗 (%s): %w", p.name, err)
			}
		}
	}

	// 各建物の南辺の開口に見える扉を置く。1マスの床の切れ目だけでは入口と分からないため、
	// 開閉できる扉エンティティで入口を明示する。扉のXが自断片に入るときだけ実体化する
	for _, b := range buildings {
		if !inFrag(b.door) {
			continue
		}
		wx := g.offsetX + (b.door - fragOrigin)
		wy := g.offsetY + (b.y + b.h - 1)
		if _, err := lifecycle.SpawnDoor(world, wx, wy, gc.DoorOrientationHorizontal); err != nil {
			return fmt.Errorf("市街地の扉配置に失敗 (x=%d, y=%d): %w", wx, wy, err)
		}
	}

	// 断片ごとの独立ストリームで敵を湧かせる。断片単位のシードなので生成順に依存しない。
	// 敵の数は規模(横幅)に比例し、大きな市街地ほど高リスク高リターンになる。
	// 種類は敵テーブルから規模を深度とみなして重み抽選する
	enemyTable, err := raw.GetEnemyTable(world.Resources.RawMaster, urbanEnemyTable)
	if err != nil {
		return fmt.Errorf("市街地の敵テーブル取得に失敗: %w", err)
	}
	fragRng := rand.New(rand.NewPCG(citySeed, uint64(fragIdx)+1))
	enemyCount := int(width-1) * (2 + fragRng.IntN(3))
	for range enemyCount {
		lx := consts.Tile(fragRng.IntN(int(g.chunkW)))
		ly := consts.Tile(fragRng.IntN(int(g.chunkH)))
		enemyName, err := raw.SelectEnemyByWeight(enemyTable, fragRng, int(width))
		if err != nil {
			return fmt.Errorf("市街地の敵抽選に失敗: %w", err)
		}
		// 壁マスに埋まる位置は湧かせない。抽選は消費済みなので決定性は保たれる
		if layout[consts.Coord[consts.Tile]{X: fragOrigin + lx, Y: ly}] == cityWall {
			continue
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
