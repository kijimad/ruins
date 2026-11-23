package overworld

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// 点在ランドマークは、集落や市街地の無い原野に小さな景色の変化を置く地物。raw の landmarks から
// 出現重みで決定的に選び、探索の単調さを崩す。

// landmarkKindAt は当選チャンクに置くランドマークの施設の id を出現重みで抽選する。
func landmarkKindAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk]) string {
	lms := raw.PtrSlice(raws.Landmarks)
	i, ok := consts.Pick(consts.HashRoll(consts.Mix2D(runSeed^placementOf(raws, rawschema.PointFeatureKindLandmark).Salt, c.X, c.Y)), lms, func(l rawschema.Landmark) float64 { return float64(l.Weight) })
	if !ok {
		// スキーマの weight>=1 と TestRealData_HasLandmarks が候補の存在を保証する
		panic("landmarkKindAt: no landmark to pick")
	}
	return lms[i].Id
}

// wildernessLandmarkFeature は自然の点在ランドマークの feature 実装。
type wildernessLandmarkFeature struct{}

// landmarkMargin はランドマークの絵とチャンクの縁や道の間に空けるマス数
const landmarkMargin consts.Tile = 2

// landmarkOrigins は幅 w 高さ h の絵の原点の候補を、チャンク内の座標で北西から順に返す。絵はチャンクの縁から
// landmarkMargin 以上離し、道のマス roads からも landmarkMargin 以上離して、道を塞がず道の脇に置く
func landmarkOrigins(w, h, chunkW, chunkH consts.Tile, roads map[consts.Coord[consts.Tile]]bool) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	for oy := landmarkMargin; oy+h+landmarkMargin <= chunkH; oy++ {
		for ox := landmarkMargin; ox+w+landmarkMargin <= chunkW; ox++ {
			if !touchesRoad(ox-landmarkMargin, oy-landmarkMargin, ox+w+landmarkMargin, oy+h+landmarkMargin, roads) {
				out = append(out, consts.Coord[consts.Tile]{X: ox, Y: oy})
			}
		}
	}
	return out
}

// touchesRoad は半開区間の矩形 [x0,x1)×[y0,y1) に道のマスがあるかを返す
func touchesRoad(x0, y0, x1, y1 consts.Tile, roads map[consts.Coord[consts.Tile]]bool) bool {
	for p := range roads {
		if p.X >= x0 && p.X < x1 && p.Y >= y0 && p.Y < y1 {
			return true
		}
	}
	return false
}

// place は当選チャンクの荒れ地に、ランドマークの施設の絵を1つ置く。景色の脇役なので主役の地物には譲り、構図を
// 壊さない。地物の優先度は chunkTypeAt が一元管理するので、ランドマークは「このチャンクの種別が
// ランドマークか」を問い合わせるだけにする。上位地物を足しても chunkTypeAt を直せば済む。
func (wildernessLandmarkFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world)) != chunkLandmark {
		return nil
	}
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return fmt.Errorf("failed to load facility templates: %w", err)
	}
	seed := consts.Mix2D(runSeed^placementOf(world.Resources.RawMaster, rawschema.PointFeatureKindLandmark).Salt, c.X, c.Y)
	// ストリーム 0 は絵と原点の抽選。spawnFacility が戦利品と敵に使う 0x4 と 0x5 と分ける
	rng := rand.New(rand.NewPCG(seed, 0))
	fac := landmarkKindAt(world.Resources.RawMaster, runSeed, c)
	cells, err := planFacility(loader, fac, rng.Uint64(), rng)
	if err != nil {
		return fmt.Errorf("failed to plan landmark %q: %w", fac, err)
	}
	roads := chunkRoadTiles(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world), g.chunkW, g.chunkH)
	origins := landmarkOrigins(consts.Tile(len(cells[0])), consts.Tile(len(cells)), g.chunkW, g.chunkH, roads)
	// 道を塞がない置き場所が無ければ、このチャンクには置かない
	if len(origins) == 0 {
		return nil
	}
	o := origins[rng.IntN(len(origins))]
	if _, _, err := spawnFacility(world, g, cells, o.X, o.Y, fac, urbanDangerAt(world, c), seed); err != nil {
		return fmt.Errorf("failed to place landmark %q: %w", fac, err)
	}
	return nil
}
