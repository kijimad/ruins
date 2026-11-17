package mapplanner

import (
	"cmp"
	"maps"
	"math"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// facingRow は向きを1つにそろえる置物の並び。sides は並びが向ける方向の候補
type facingRow struct {
	members, sides []consts.Coord[consts.Tile]
}

// RowFacings は facing の置物ごとの向きを返す。隣り合う置物は東西か南北のまっすぐな並びにまとめ、
// 並びごとに1つの向きを決めるので、端の1台だけ向きがずれることは無い。一番長い並びから1本ずつ決め、
// 決めた置物を除いた残りから並びを作り直す。L字やT字で交わる置物は長い並びの向きにそろい、塊の残りも並びとして向く。
// 並びと直角の2方向のうち、背の塞がりの重み weight が大きく前が小さい側を向く。同じなら南・東・西・北の順で先の側を向く。
// 並びの1台でも背が塞がれば、並び全体を背へ寄せる。軸に沿って隣り合う同じ向きの2台は必ず同じ並びで決まるので、
// 寄せる位置も同じになり、端を省き合っても継ぎ目に隙間が出ない。
// 1台だけの置物は4方向から同じく選ぶ
func RowFacings(facers map[consts.Coord[consts.Tile]]bool, weight func(consts.Coord[consts.Tile]) float64) map[consts.Coord[consts.Tile]]gc.Facing {
	order := slices.SortedFunc(maps.Keys(facers), func(a, b consts.Coord[consts.Tile]) int { return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X)) })
	left := maps.Clone(facers)
	facings := map[consts.Coord[consts.Tile]]gc.Facing{}
	decide := func(row facingRow) {
		best, bestScore, back := row.sides[0], math.Inf(-1), 0.0
		for _, d := range row.sides {
			score, b := 0.0, 0.0
			for _, m := range row.members {
				b += weight(m.Sub(d))
				score += weight(m.Sub(d)) - weight(m.Add(d))
			}
			if score > bestScore {
				best, bestScore, back = d, score, b
			}
		}
		for _, m := range row.members {
			facings[m] = gc.Facing{Dir: best, Backed: back > 0}
			delete(left, m)
		}
	}
	for len(left) > 0 {
		row := longestRow(order, left)
		if len(row.members) > 1 {
			decide(row)
			continue
		}
		// 一番長い並びが1台なら残りはどれも1台なので、並びを探し直さずに順に決める
		for _, c := range order {
			if left[c] {
				decide(facingRow{members: []consts.Coord[consts.Tile]{c}, sides: aloneSides})
			}
		}
	}
	return facings
}

var (
	// rowAxes は並びの軸。東西の並びを南北より先に探す
	rowAxes = []struct{ back, ahead consts.Coord[consts.Tile] }{{consts.TileWest, consts.TileEast}, {consts.TileNorth, consts.TileSouth}}
	// eastWestRowSides、northSouthRowSides、aloneSides は向きの候補。同点なら先の側を向く
	eastWestRowSides   = []consts.Coord[consts.Tile]{consts.TileSouth, consts.TileNorth}
	northSouthRowSides = []consts.Coord[consts.Tile]{consts.TileEast, consts.TileWest}
	aloneSides         = []consts.Coord[consts.Tile]{consts.TileSouth, consts.TileEast, consts.TileWest, consts.TileNorth}
)

// longestRow は残りの置物 left の中で一番長いまっすぐな並びを返す。order は置物を北西から並べた順。
// 同じ長さなら北西に近い置物から始まる並び、東西の並びを先にする。並びが無ければ1台だけの並びを返す
func longestRow(order []consts.Coord[consts.Tile], left map[consts.Coord[consts.Tile]]bool) facingRow {
	var best facingRow
	for _, c := range order {
		if !left[c] {
			continue
		}
		if best.members == nil {
			best = facingRow{members: []consts.Coord[consts.Tile]{c}}
		}
		for _, axis := range rowAxes {
			if left[c.Add(axis.back)] || !left[c.Add(axis.ahead)] {
				continue
			}
			row := facingRow{sides: eastWestRowSides}
			if axis.back == consts.TileNorth {
				row.sides = northSouthRowSides
			}
			for m := c; left[m]; m = m.Add(axis.ahead) {
				row.members = append(row.members, m)
			}
			if len(row.members) > len(best.members) {
				best = row
			}
		}
	}
	return best
}
