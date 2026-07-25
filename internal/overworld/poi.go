package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// 自然の点在POIは、集落や市街地の無い原野に小さな景色の変化を置く地物。廃屋・農家跡・
// 祠・キャンプ跡を重み抽選し、探索の単調さを崩す。v1 は構造と prop だけを持ち、
// 戦利品やイベントはアイテム・イベント設計が固まってから続ける。

// poiPlacement は点在POIのリージョン配置。地物の中では最も密に置く。
var poiPlacement = Placement{Spacing: 3, Separation: 1, Salt: poiSalt}

// wildernessPOIFeature は自然の点在POIの feature 実装。
type wildernessPOIFeature struct{}

// place は当選チャンクの荒れ地に小構造物を1つ置く。景色の脇役なので主役の地物には譲り、構図を
// 壊さない。地物の優先度は chunkTypeAt が一元管理するので、POI は「このチャンクの種別が POI か」を
// 問い合わせるだけにする。上位地物を足しても chunkTypeAt を直せば済み、POI 側の変更は要らない。
func (wildernessPOIFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(runSeed, c, rows) != chunkPOI {
		return nil
	}

	rng := rand.New(rand.NewPCG(ChunkSeed2D(runSeed^poiSalt, c.X, c.Y), 0))
	// 構造物がチャンク境界をはみ出さないよう内側に収める。最大の小屋 7x5 ぶんの余地を残し、
	// 残り範囲でずらす。小さなチャンクでも境界越えの残留を作らない
	const margin, maxHut = 2, 8
	spanX := max(1, int(g.chunkW)-2*margin-maxHut)
	spanY := max(1, int(g.chunkH)-2*margin-maxHut)
	ox := g.offsetX + consts.Tile(margin+rng.IntN(spanX))
	oy := g.offsetY + consts.Tile(margin+rng.IntN(spanY))
	origin := consts.Coord[consts.Tile]{X: ox, Y: oy}

	roll := rng.IntN(100)
	switch {
	case roll < 30: // 廃屋。生活の跡が残る小屋
		return stampHut(world, g, rng, origin, 6, 5, []string{"bed", "closet"})
	case roll < 55: // 農家跡。納屋に物資の跡
		return stampHut(world, g, rng, origin, 7, 5, []string{"barrel", "crate", "茶色い樽"})
	case roll < 75: // 祠。石柱と蝋燭だけの露天の構造物
		return spawnPOIProps(world, ox, oy, []townSpot{
			{"stone_pillar", 0, 0},
			{"candle", -1, 1},
			{"candle", 1, 1},
		})
	default: // キャンプ跡。誰かが夜を越した跡
		return spawnPOIProps(world, ox, oy, []townSpot{
			{"bonfire", 0, 0},
			{"crate", 1, 1},
			{"bench", -1, 0},
		})
	}
}

// stampHut は外周壁・内側床・南辺出入口の小屋を置き、内装 prop を屋内へ順に配置する。
// 市街地の街区と同じ構法だが、単チャンク完結なので断片クリップは不要。
func stampHut(world w.World, g chunkGeom, rng *rand.Rand, origin consts.Coord[consts.Tile], hw, hh consts.Tile, props []string) error {
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
			if err := replaceTile(world, tiles, lx, ly, name); err != nil {
				return fmt.Errorf("POI小屋の配置に失敗 (x=%d, y=%d): %w", lx, ly, err)
			}
		}
	}
	for i, name := range props {
		// 屋内の北側の壁沿いへ左から順に並べる。出入口の導線と重ねない
		pos := consts.Coord[consts.Tile]{X: ox + 1 + consts.Tile(i), Y: oy + 1}
		if _, err := lifecycle.SpawnProp(world, name, pos.X, pos.Y); err != nil {
			return fmt.Errorf("POI内装の配置に失敗 (%s): %w", name, err)
		}
	}
	// 南辺の開口に見える扉を置く。壁の切れ目だけだと原野の中の謎の壁に見えるため、
	// 廃屋としての入口を明示する。南壁は東西に走るので向きは Vertical
	if _, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: door, Y: oy + hh - 1}, gc.DoorOrientationVertical); err != nil {
		return fmt.Errorf("POI小屋の扉配置に失敗: %w", err)
	}
	return nil
}

// spawnPOIProps は露天POIの prop 一式を基準座標からの相対で配置する。
func spawnPOIProps(world w.World, ox, oy consts.Tile, spots []townSpot) error {
	for _, s := range spots {
		if _, err := lifecycle.SpawnProp(world, s.name, ox+s.dx, oy+s.dy); err != nil {
			return fmt.Errorf("POIの配置に失敗 (%s): %w", s.name, err)
		}
	}
	return nil
}
