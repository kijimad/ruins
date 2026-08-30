package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// 点在ランドマークは、集落や市街地の無い原野に小さな景色の変化を置く地物。廃屋・農家跡・
// 祠・キャンプ跡を決定的に選び、探索の単調さを崩す。今は構造と prop だけを持ち、
// 戦利品やイベントはアイテム・イベント設計が固まってから足す。

// landmarkPlacement は点在ランドマークのリージョン配置。地物の中では最も密に置く。
var landmarkPlacement = Placement{Spacing: 3, Separation: 1, Salt: landmarkSalt}

// landmarkKind は点在ランドマークの種別。地図の記号と実際に置く構造の両方をこの種別で決める。
type landmarkKind string

const (
	landmarkAbandonedHut landmarkKind = "abandoned_hut" // 廃屋。生活の跡が残る小屋
	landmarkFarmstead    landmarkKind = "farmstead"     // 農家跡。納屋に物資の跡
	landmarkShrine       landmarkKind = "shrine"        // 祠。石柱と蝋燭だけの露天の構造物
	landmarkCampsite     landmarkKind = "campsite"      // キャンプ跡。誰かが夜を越した跡
)

// landmarkKindAt は当選チャンクに置くランドマークの種別を返す純関数。地図の記号と生成の構造が
// 同じ種別を引くので、俯瞰図の見た目と実体が食い違わない。settlementVillageRoll と同じく
// (runSeed, 座標) だけで決まる。出現率は廃屋30・農家跡25・祠20・キャンプ跡25。
func landmarkKindAt(runSeed uint64, c consts.Coord[consts.Chunk]) landmarkKind {
	switch roll := ChunkSeed2D(runSeed^landmarkSalt, c.X, c.Y) % 100; {
	case roll < 30:
		return landmarkAbandonedHut
	case roll < 55:
		return landmarkFarmstead
	case roll < 75:
		return landmarkShrine
	default:
		return landmarkCampsite
	}
}

// landmarkPlaceType は種別を地図の表示分類へ写す。記号と凡例名は placeGlyphs が一元管理する。
func landmarkPlaceType(k landmarkKind) placeType {
	switch k {
	case landmarkAbandonedHut:
		return placeAbandonedHut
	case landmarkFarmstead:
		return placeFarmstead
	case landmarkShrine:
		return placeShrine
	case landmarkCampsite:
		return placeCampsite
	}
	panic("unknown landmarkKind: " + string(k))
}

// wildernessLandmarkFeature は自然の点在ランドマークの feature 実装。
type wildernessLandmarkFeature struct{}

// place は当選チャンクの荒れ地に小構造物を1つ置く。景色の脇役なので主役の地物には譲り、構図を
// 壊さない。地物の優先度は chunkTypeAt が一元管理するので、ランドマークは「このチャンクの種別が
// ランドマークか」を問い合わせるだけにする。上位地物を足しても chunkTypeAt を直せば済む。
func (wildernessLandmarkFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(runSeed, c, rows) != chunkLandmark {
		return nil
	}

	rng := rand.New(rand.NewPCG(ChunkSeed2D(runSeed^landmarkSalt, c.X, c.Y), 0))
	// 構造物がチャンク境界をはみ出さないよう内側に収める。maxHut は最大の小屋の幅 7 に壁1枚分の
	// 余白を足した値で、原点をどこへずらしても外壁と南辺の扉が境界へ接しない
	const margin, maxHut = 2, 8
	spanX := max(1, int(g.chunkW)-2*margin-maxHut)
	spanY := max(1, int(g.chunkH)-2*margin-maxHut)
	ox := g.offsetX + consts.Tile(margin+rng.IntN(spanX))
	oy := g.offsetY + consts.Tile(margin+rng.IntN(spanY))
	origin := consts.Coord[consts.Tile]{X: ox, Y: oy}

	kind := landmarkKindAt(runSeed, c)
	switch kind {
	case landmarkAbandonedHut:
		return drawHut(world, g, rng, origin, 6, 5, []string{"bed", "closet"})
	case landmarkFarmstead:
		return drawHut(world, g, rng, origin, 7, 5, []string{"barrel", "crate"})
	case landmarkShrine:
		return spawnLandmarkProps(world, origin, []relSpot{
			{"stone_pillar", 0, 0},
			{"candle", -1, 1},
			{"candle", 1, 1},
		})
	case landmarkCampsite:
		return spawnLandmarkProps(world, origin, []relSpot{
			{"hearth", 0, 0},
			{"bonfire", 0, 0}, // 石組の上に点いた火を重ねる
			{"crate", 1, 1},
			{"bench", -1, 0},
		})
	}
	panic("unknown landmarkKind: " + string(kind))
}

// drawHut は外周壁・内側床・南辺出入口の小屋を置き、内装 prop を屋内へ順に配置する。
// 市街地の街区と同じ構法だが、単チャンク完結なので断片クリップは不要。
func drawHut(world w.World, g chunkGeom, rng *rand.Rand, origin consts.Coord[consts.Tile], hw, hh consts.Tile, props []string) error {
	tiles := g.tiles.get()
	ox, oy := origin.X, origin.Y
	door := ox + 1 + consts.Tile(rng.IntN(int(hw-2)))
	for ly := oy; ly < oy+hh; ly++ {
		for lx := ox; lx < ox+hw; lx++ {
			name := consts.TileNameFloor
			perimeter := lx == ox || lx == ox+hw-1 || ly == oy || ly == oy+hh-1
			if perimeter && (ly != oy+hh-1 || lx != door) {
				name = consts.TileNameDWall
			}
			if err := replaceTile(world, tiles, consts.Coord[consts.Tile]{X: lx, Y: ly}, name); err != nil {
				return fmt.Errorf("failed to place landmark hut (x=%d, y=%d): %w", lx, ly, err)
			}
		}
	}
	for i, name := range props {
		// 屋内の北側の壁沿いへ左から順に並べる。出入口の導線と重ねない
		pos := consts.Coord[consts.Tile]{X: ox + 1 + consts.Tile(i), Y: oy + 1}
		if _, err := lifecycle.SpawnProp(world, name, pos.X, pos.Y); err != nil {
			return fmt.Errorf("failed to place landmark interior prop (%s): %w", name, err)
		}
	}
	// 南辺の開口に見える扉を置く。壁の切れ目だけだと原野の中の謎の壁に見えるため、
	// 廃屋としての入口を明示する。南壁は東西に走るので向きは Vertical
	if _, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: door, Y: oy + hh - 1}, gc.DoorOrientationVertical); err != nil {
		return fmt.Errorf("failed to place landmark hut door: %w", err)
	}
	return nil
}

// spawnLandmarkProps は露天ランドマークの prop 一式を基準座標からの相対で配置する。
func spawnLandmarkProps(world w.World, origin consts.Coord[consts.Tile], spots []relSpot) error {
	for _, s := range spots {
		if _, err := lifecycle.SpawnProp(world, s.name, origin.X+s.dx, origin.Y+s.dy); err != nil {
			return fmt.Errorf("failed to place landmark prop (%s): %w", s.name, err)
		}
	}
	return nil
}
