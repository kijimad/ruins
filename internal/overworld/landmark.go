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

// 点在ランドマークは、集落や市街地の無い原野に小さな景色の変化を置く地物。廃屋・農家跡・
// 祠・キャンプ跡を決定的に選び、探索の単調さを崩す。

// landmarkKindAt は当選チャンクに置くランドマークの施設の id を出現重みで抽選する。
func landmarkKindAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk]) string {
	lms := raw.PtrSlice(raws.Landmarks)
	i, ok := consts.Pick(consts.HashRoll(ChunkSeed2D(runSeed^landmarkSalt, c.X, c.Y)), lms, func(l rawschema.Landmark) float64 { return float64(l.Weight) })
	if !ok {
		// スキーマの weight>=1 と TestRealData_HasLandmarks が候補の存在を保証する
		panic("landmarkKindAt: no landmark to pick")
	}
	return lms[i].Id
}

// wildernessLandmarkFeature は自然の点在ランドマークの feature 実装。
type wildernessLandmarkFeature struct{}

// landmarkMargin はランドマークの絵とチャンクの縁の間に空けるマス数
const landmarkMargin consts.Tile = 2

// place は当選チャンクの荒れ地に、ランドマークの施設の絵を1つ置く。景色の脇役なので主役の地物には譲り、構図を
// 壊さない。地物の優先度は chunkTypeAt が一元管理するので、ランドマークは「このチャンクの種別が
// ランドマークか」を問い合わせるだけにする。上位地物を足しても chunkTypeAt を直せば済む。
func (wildernessLandmarkFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	if chunkTypeAt(world.Resources.RawMaster, runSeed, c, cols) != chunkLandmark {
		return nil
	}
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return fmt.Errorf("failed to load facility templates: %w", err)
	}
	seed := ChunkSeed2D(runSeed^landmarkSalt, c.X, c.Y)
	rng := rand.New(rand.NewPCG(seed, 0))
	fac := landmarkKindAt(world.Resources.RawMaster, runSeed, c)
	cells, err := planFacility(loader, fac, rng.Uint64(), rng)
	if err != nil {
		return fmt.Errorf("failed to plan landmark %q: %w", fac, err)
	}
	// 絵がチャンクの縁から landmarkMargin 以上離れる範囲で原点を引く
	spanX := max(1, int(g.chunkW-2*landmarkMargin)-len(cells[0])+1)
	spanY := max(1, int(g.chunkH-2*landmarkMargin)-len(cells)+1)
	ox := landmarkMargin + consts.Tile(rng.IntN(spanX))
	oy := landmarkMargin + consts.Tile(rng.IntN(spanY))
	if _, _, err := spawnFacility(world, g, cells, ox, oy, fac, urbanDangerAt(world, c), seed); err != nil {
		return fmt.Errorf("failed to place landmark %q: %w", fac, err)
	}
	return nil
}
