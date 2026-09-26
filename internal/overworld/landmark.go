package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// 点在ランドマークは、集落や市街地の無い原野に小さな景色の変化を置く地物。廃屋・農家跡・
// 祠・キャンプ跡を決定的に選び、探索の単調さを崩す。種別・出現重み・小屋寸法・prop は raw.toml の
// landmarks 行が単一出典で、Go に残るのはデータ化できない描画関数 drawers だけ。

// landmarkKindAt は当選チャンクに置くランドマークの id を出現重みで抽選する純関数。地図の記号と生成の
// 構造が同じ id を引くので、俯瞰図の見た目と実体が食い違わない。重みは raw.toml の landmarks 行から引く。
func landmarkKindAt(raws oapi.Raws, runSeed uint64, c consts.Coord[consts.Chunk]) string {
	lms := raw.PtrSlice(raws.Landmarks)
	total := 0
	for _, l := range lms {
		total += int(l.Weight)
	}
	// landmarks が空だと total==0 で IntN が壊れる。validate が非空を保証する
	roll := int(ChunkSeed2D(runSeed^landmarkSalt, c.X, c.Y) % uint64(total))
	for _, l := range lms {
		roll -= int(l.Weight)
		if roll < 0 {
			return l.Id
		}
	}
	return lms[len(lms)-1].Id
}

// wildernessLandmarkFeature は自然の点在ランドマークの feature 実装。
type wildernessLandmarkFeature struct{}

// landmarkDrawer はランドマーク1種を描く関数。データ化できない幾何なので Go に残す。
type landmarkDrawer func(world w.World, g chunkGeom, rng *rand.Rand, origin consts.Coord[consts.Tile], lm oapi.Landmark) error

// drawers は drawer キーから描画関数を引く単一出典。新しい描画を足すときだけここへ1行足し、tsp の
// DrawerKey enum にも同じキーを加える。両者の一致は被覆テストで固定する。
var drawers = map[oapi.DrawerKey]landmarkDrawer{
	oapi.Hut:  drawHutLandmark,
	oapi.Open: drawOpenLandmark,
}

// drawHutLandmark は landmarks 行の小屋寸法と prop 名から外周壁の小屋を描く。prop は北壁沿いに順に並べる。
func drawHutLandmark(world w.World, g chunkGeom, rng *rand.Rand, origin consts.Coord[consts.Tile], lm oapi.Landmark) error {
	names := make([]string, len(lm.Props))
	for i, p := range lm.Props {
		names[i] = p.Name
	}
	return drawHut(world, g, rng, origin, consts.Tile(lm.HutW), consts.Tile(lm.HutH), names)
}

// drawOpenLandmark は landmarks 行の prop を相対座標で露天に置く。
func drawOpenLandmark(world w.World, _ chunkGeom, _ *rand.Rand, origin consts.Coord[consts.Tile], lm oapi.Landmark) error {
	spots := make([]relSpot, len(lm.Props))
	for i, p := range lm.Props {
		spots[i] = relSpot{name: p.Name, dx: consts.Tile(p.Dx), dy: consts.Tile(p.Dy)}
	}
	return spawnLandmarkProps(world, origin, spots)
}

// place は当選チャンクの荒れ地に小構造物を1つ置く。景色の脇役なので主役の地物には譲り、構図を
// 壊さない。地物の優先度は chunkTypeAt が一元管理するので、ランドマークは「このチャンクの種別が
// ランドマークか」を問い合わせるだけにする。上位地物を足しても chunkTypeAt を直せば済む。
func (wildernessLandmarkFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(runSeed, c, cols) != chunkLandmark {
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

	id := landmarkKindAt(world.Resources.RawMaster, runSeed, c)
	lm, ok := raw.GetLandmark(world.Resources.RawMaster, id)
	if !ok {
		return fmt.Errorf("landmark %q not found", id)
	}
	draw, ok := drawers[lm.Drawer]
	if !ok {
		return fmt.Errorf("landmark %q drawer %q not registered", id, lm.Drawer)
	}
	return draw(world, g, rng, origin, lm)
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
			if _, err := replaceTile(world, tiles, consts.Coord[consts.Tile]{X: lx, Y: ly}, name); err != nil {
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
