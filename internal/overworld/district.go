package overworld

import (
	"hash/fnv"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// district は区域の1種類。raw の districts の行から作る
type district struct {
	row  rawschema.District
	salt uint64
}

// districtRegion は区域の1つの当選。左上 anchor から東と南へ w×h チャンク広がる
type districtRegion struct {
	layout rawschema.DistrictLayout
	anchor consts.Coord[consts.Chunk]
	w, h   consts.Chunk
	// seed は区域の中の地区と施設を引く種
	seed uint64
}

// districtsFrom は raw の districts の行を、行の順に区域の種類にする
func districtsFrom(raws rawschema.Raws) []district {
	rows := raw.PtrSlice(raws.Districts)
	out := make([]district, len(rows))
	for i, r := range rows {
		h := fnv.New64a()
		_, _ = h.Write([]byte(r.Id))
		out[i] = district{row: r, salt: districtSalt ^ h.Sum64()}
	}
	return out
}

// regionIn はリージョン ry の区域を返す。大きさを重みで引き、区域がリージョンの縦の範囲
// [ry*spacing, (ry+1)*spacing - separation) に収まるよう位置を引く。区域が自分のリージョンから
// はみ出さないので、チャンクを含む区域はそのチャンクの行が属するリージョンだけを見れば決まる
func (d district) regionIn(runSeed uint64, ry, cols consts.Chunk) districtRegion {
	rng := rand.New(rand.NewPCG(ChunkSeed2D(runSeed^d.salt, ry, 0), 0))
	i, ok := consts.Pick(rng, d.row.Sizes, func(s rawschema.DistrictSize) float64 { return float64(s.Weight) })
	if !ok {
		// 大きさの候補が1つ以上あり重みが1以上であることを raw のスキーマが保証する
		panic("district has no size: " + d.row.Id)
	}
	size := d.row.Sizes[i]
	w := min(consts.Chunk(size.W), cols)
	anchor := consts.Coord[consts.Chunk]{
		X: consts.Chunk(rng.IntN(int(cols-w) + 1)),
		Y: ry*consts.Chunk(d.row.Spacing) + consts.Chunk(rng.IntN(d.row.Spacing-d.row.Separation-size.H+1)),
	}
	return districtRegion{
		layout: d.row.Layout,
		anchor: anchor,
		w:      w,
		h:      consts.Chunk(size.H),
		seed:   ChunkSeed2D(runSeed^d.salt, anchor.X, anchor.Y),
	}
}

// regionAt は c の行が属するリージョンの区域を返す。区域が c を含まなければ ok=false
func (d district) regionAt(runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) (districtRegion, bool) {
	r := d.regionIn(runSeed, consts.FloorDiv(c.Y, consts.Chunk(d.row.Spacing)), cols)
	return r, r.contains(c)
}

// districtAt は c を含む区域を返す。種類の違う区域が重なれば、raw の行の順で先の種類を残し、後の種類の区域は
// 丸ごと出さない。街路の途中に塀が刺さった半端な区域を作らないため
func districtAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) (districtRegion, bool) {
	ds := districtsFrom(raws)
	for i, d := range ds {
		r, ok := d.regionAt(runSeed, c, cols)
		if ok && !overlapsAny(r, ds[:i], runSeed, cols) {
			return r, true
		}
	}
	return districtRegion{}, false
}

// overlapsAny は区域 r が、種類 higher のどれかの区域と重なるかを返す。r の行が属するリージョンだけを見る
func overlapsAny(r districtRegion, higher []district, runSeed uint64, cols consts.Chunk) bool {
	for _, d := range higher {
		spacing := consts.Chunk(d.row.Spacing)
		for ry := consts.FloorDiv(r.anchor.Y, spacing); ry <= consts.FloorDiv(r.anchor.Y+r.h-1, spacing); ry++ {
			if d.regionIn(runSeed, ry, cols).overlaps(r) {
				return true
			}
		}
	}
	return false
}

// overlaps は2つの区域が1チャンクでも重なるかを返す
func (r districtRegion) overlaps(o districtRegion) bool {
	return r.anchor.X < o.anchor.X+o.w && o.anchor.X < r.anchor.X+r.w && r.anchor.Y < o.anchor.Y+o.h && o.anchor.Y < r.anchor.Y+r.h
}

// contains は c が区域の中かを返す
func (r districtRegion) contains(c consts.Coord[consts.Chunk]) bool {
	return c.X >= r.anchor.X && c.X < r.anchor.X+r.w && c.Y >= r.anchor.Y && c.Y < r.anchor.Y+r.h
}

// local は c の区域の中の座標を返す
func (r districtRegion) local(c consts.Coord[consts.Chunk]) consts.Coord[consts.Chunk] {
	return c.Sub(r.anchor)
}

// span は区域の長い方の辺のチャンク数。施設の規模の足切りと敵の数に使う
func (r districtRegion) span() consts.Chunk {
	return max(r.w, r.h)
}

// distance は c から区域までのチェビシェフ距離。区域の中なら0
func (r districtRegion) distance(c consts.Coord[consts.Chunk]) consts.Chunk {
	dx := max(r.anchor.X-c.X, 0, c.X-(r.anchor.X+r.w-1))
	dy := max(r.anchor.Y-c.Y, 0, c.Y-(r.anchor.Y+r.h-1))
	return max(dx, dy)
}
