package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 道のチャンクの中の道の形。道がどのチャンクを通りどの縁へ抜けるかは区画の計画 road_plan.go が決め、ここは出入り口から
// チャンクの中に描く道の辺を作る。舗装 road.go と散布の回避 scatter.go が同じ辺を使う。

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

// lanedPort は縁に沿った位置を決めた出入り口。at はチャンクの中の座標
type lanedPort struct {
	dir RoadDir
	at  consts.Tile
}

// roadSegsAt は c が道のチャンクなら、チャンクの中に描く道の辺と道の行を返す。出入り口の位置は、縁の中央、区域の門の通りの真ん中、
// 隣の集落の施設の玄関のどれかにする
func roadSegsAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk, chunkW, chunkH consts.Tile) ([]roadSeg, rawschema.Road, bool, error) {
	ports, road, ok := roadAt(raws, runSeed, c, cols, spawnRow)
	if !ok {
		return nil, rawschema.Road{}, false, nil
	}
	laned := make([]lanedPort, 0, len(ports))
	for _, p := range ports {
		size := chunkW
		if p.dir == RoadW || p.dir == RoadE {
			size = chunkH
		}
		at := size / 2
		switch p.lane {
		case laneCenter:
		case laneGate:
			// 門は区域の西の通りの帯に開く
			at = urbanStreetW / 2
		case laneEntrance:
			l, err := settlementLayoutAt(raws, runSeed, c.Add(stepOf(p.dir)), cols, spawnRow, chunkW, chunkH)
			if err != nil {
				return nil, rawschema.Road{}, false, err
			}
			at = l.entrance
		}
		laned = append(laned, lanedPort{dir: p.dir, at: at})
	}
	return roadChunkSegs(c, laned, road.Width, chunkW, chunkH), road, true, nil
}

// roadChunkSegs は道のチャンク c の中に描く道の辺を、絶対タイル座標で返す。各出入り口の位置から縁に垂直に中心の行か列まで進み、
// 中心まで折れる。各辺は中心にある幅 width の四角まで含めるので、折れる所や交わる所が欠けない
func roadChunkSegs(c consts.Coord[consts.Chunk], ports []lanedPort, width int, chunkW, chunkH consts.Tile) []roadSeg {
	wd := consts.Tile(width)
	lo, hi := wd/2, wd-wd/2-1
	x0, y0 := c.X.Tiles(chunkW), c.Y.Tiles(chunkH)
	cx, cy := x0+chunkW/2, y0+chunkH/2
	var segs []roadSeg
	for _, p := range ports {
		switch p.dir {
		case RoadN, RoadS:
			x := x0 + p.at
			if p.dir == RoadN {
				segs = append(segs, span(orientVertical, x, y0, cy+hi))
			} else {
				segs = append(segs, span(orientVertical, x, cy-lo, y0+chunkH-1))
			}
			if x != cx {
				segs = append(segs, span(orientHorizontal, cy, min(x, cx)-lo, max(x, cx)+hi))
			}
		case RoadW, RoadE:
			y := y0 + p.at
			if p.dir == RoadW {
				segs = append(segs, span(orientHorizontal, y, x0, cx+hi))
			} else {
				segs = append(segs, span(orientHorizontal, y, cx-lo, x0+chunkW-1))
			}
			if y != cy {
				segs = append(segs, span(orientVertical, cx, min(y, cy)-lo, max(y, cy)+hi))
			}
		}
	}
	return segs
}

// portDirs は出入り口の方角を集める
func portDirs(ports []roadPort) RoadDir {
	var d RoadDir
	for _, p := range ports {
		d |= p.dir
	}
	return d
}
