package overworld

import (
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 道の幾何の唯一の出典。道は隣接リージョンの当選集落の道ばたのチャンクどうしをチャンク単位で結び、通るチャンクを
// 道のチャンクにする。道のチャンクは道だけが使うので、施設や区域と道は重ならない。チャンクの種別 classifyChunk、
// 舗装 road.go、散布の回避 scatter.go、マクロ地図 road_overlay.go がここの経路を共有する。

// roadOrient は道の一辺の向き。実体は文字列で、%v やテスト失敗の出力に種別名が出る。
type roadOrient string

const (
	orientHorizontal roadOrient = "horizontal" // 行を固定して列方向へ伸びる
	orientVertical   roadOrient = "vertical"   // 列を固定して行方向へ伸びる
)

// roadSeg は道の一辺。座標は絶対タイルで、fixed は道の中心線。
type roadSeg struct {
	orient roadOrient
	fixed  consts.Tile // orient=horizontal なら固定する Y、vertical なら固定する X
	lo, hi consts.Tile // 可変軸の範囲。lo<=hi で端点を含む
}

// span は lo と hi を小さい順に並べた辺を作る
func span(orient roadOrient, fixed, a, b consts.Tile) roadSeg {
	return roadSeg{orient: orient, fixed: fixed, lo: min(a, b), hi: max(a, b)}
}

// roadsideOf は集落 s の脇を道が通るチャンクを返す。西隣にし、西端の列の集落だけ東隣にする。集落ごとに1つに決まるので、
// 北から来る道と南へ出る道が同じチャンクを通り、集落の脇を過ぎる。列が1つしかなければ脇が無いので ok=false
func roadsideOf(s consts.Coord[consts.Chunk], cols consts.Chunk) (consts.Coord[consts.Chunk], bool) {
	switch {
	case s.X > 0:
		return consts.Coord[consts.Chunk]{X: s.X - 1, Y: s.Y}, true
	case cols > 1:
		return consts.Coord[consts.Chunk]{X: s.X + 1, Y: s.Y}, true
	}
	return consts.Coord[consts.Chunk]{}, false
}

// linkChunks は集落 a から b への道が通るチャンクを、a の道ばたから b の道ばたまで順に返す。道ばたから縦に進み、
// 2つの集落の間の行で横へ折れ、相手の道ばたへ縦に進む。折れる行が両端の集落の行と違えば集落を通らない。
// 間に行が無いときは外側の行で折れ、それでも集落を通るなら道を作らない
func linkChunks(a, b consts.Coord[consts.Chunk], cols consts.Chunk) []consts.Coord[consts.Chunk] {
	ra, okA := roadsideOf(a, cols)
	rb, okB := roadsideOf(b, cols)
	if !okA || !okB {
		return nil
	}
	top, bottom := min(a.Y, b.Y), max(a.Y, b.Y)
	var turns []consts.Chunk
	if bottom-top >= 2 {
		turns = append(turns, (top+bottom)/2)
	}
	turns = append(turns, top-1, bottom+1)
	for _, m := range turns {
		path := chunkLine(nil, ra, consts.Coord[consts.Chunk]{X: ra.X, Y: m})
		path = chunkLine(path, consts.Coord[consts.Chunk]{X: ra.X, Y: m}, consts.Coord[consts.Chunk]{X: rb.X, Y: m})
		path = chunkLine(path, consts.Coord[consts.Chunk]{X: rb.X, Y: m}, rb)
		if !slices.Contains(path, a) && !slices.Contains(path, b) {
			return path
		}
	}
	return nil
}

// chunkLine は path の後ろへ from から to までの縦か横の直線のチャンクを足して返す。path の末尾が from なら重ねない
func chunkLine(path []consts.Coord[consts.Chunk], from, to consts.Coord[consts.Chunk]) []consts.Coord[consts.Chunk] {
	step := consts.Coord[consts.Chunk]{X: sign(to.X - from.X), Y: sign(to.Y - from.Y)}
	for p := from; ; p = p.Add(step) {
		if len(path) == 0 || path[len(path)-1] != p {
			path = append(path, p)
		}
		if p == to {
			return path
		}
	}
}

// sign は v の符号を -1, 0, 1 で返す
func sign(v consts.Chunk) consts.Chunk {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// roadLink は道で結ぶ2つの当選と、その道
type roadLink struct {
	a, b consts.Coord[consts.Chunk]
	road rawschema.Road
}

// crossingRoads はチャンク c を横切りうる道を返す。道を持つ地物ごとに、c の行が属するリージョン r の
// (r-1,r) と (r,r+1) の当選を結ぶ2本だけが c を横切りうる
func crossingRoads(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) []roadLink {
	var links []roadLink
	for _, f := range raw.PtrSlice(raws.PointFeatures) {
		if f.Road == nil {
			continue
		}
		p := placementFrom(f)
		r := consts.FloorDiv(c.Y, p.Spacing)
		for pr := r - 1; pr <= r; pr++ {
			links = append(links, roadLink{a: p.WinnerOf(runSeed, pr, cols), b: p.WinnerOf(runSeed, pr+1, cols), road: *f.Road})
		}
	}
	return links
}

// onRoad は c を道が通るかを返す。チャンクの種別の分類が道のチャンクかを決めるのに使う
func onRoad(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) bool {
	for _, link := range crossingRoads(raws, runSeed, c, cols) {
		if slices.Contains(linkChunks(link.a, link.b, cols), c) {
			return true
		}
	}
	return false
}

// roadDirsAt は道のチャンク c から道がつながる方角と、その道を返す。経路の前後のチャンクのうち、道のチャンクに
// 分類された隣へだけつなぐので、集落や市街地に経路がぶつかる所で道が途切れる。道のチャンクでなければ ok=false
func roadDirsAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk) (RoadDir, rawschema.Road, bool) {
	if chunkTypeAt(raws, runSeed, c, cols, spawnRow) != chunkRoad {
		return 0, rawschema.Road{}, false
	}
	var dirs RoadDir
	var road rawschema.Road
	for _, link := range crossingRoads(raws, runSeed, c, cols) {
		path := linkChunks(link.a, link.b, cols)
		i := slices.Index(path, c)
		if i < 0 {
			continue
		}
		road = link.road
		for _, j := range []int{i - 1, i + 1} {
			if j < 0 || j >= len(path) || chunkTypeAt(raws, runSeed, path[j], cols, spawnRow) != chunkRoad {
				continue
			}
			dirs |= roadDirToward(c, path[j])
		}
	}
	return dirs, road, true
}

// roadDirToward は c から隣のチャンク n への方角を返す
func roadDirToward(c, n consts.Coord[consts.Chunk]) RoadDir {
	switch {
	case n.Y < c.Y:
		return RoadN
	case n.Y > c.Y:
		return RoadS
	case n.X > c.X:
		return RoadE
	default:
		return RoadW
	}
}

// roadChunkSegs は道のチャンク c の中に描く道の辺を、絶対タイル座標で返す。チャンクの中心から dirs の方角の縁まで伸ばす。
// 各辺は中心にある幅 width の四角まで含めるので、折れる所や交わる所が欠けない
func roadChunkSegs(c consts.Coord[consts.Chunk], dirs RoadDir, width int, chunkW, chunkH consts.Tile) []roadSeg {
	wd := consts.Tile(width)
	x0, y0 := c.X.Tiles(chunkW), c.Y.Tiles(chunkH)
	cx, cy := x0+chunkW/2, y0+chunkH/2
	var segs []roadSeg
	if dirs&RoadN != 0 {
		segs = append(segs, span(orientVertical, cx, y0, cy+wd-wd/2-1))
	}
	if dirs&RoadS != 0 {
		segs = append(segs, span(orientVertical, cx, cy-wd/2, y0+chunkH-1))
	}
	if dirs&RoadW != 0 {
		segs = append(segs, span(orientHorizontal, cy, x0, cx+wd-wd/2-1))
	}
	if dirs&RoadE != 0 {
		segs = append(segs, span(orientHorizontal, cy, cx-wd/2, x0+chunkW-1))
	}
	return segs
}
