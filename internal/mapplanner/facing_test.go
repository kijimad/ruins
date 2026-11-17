package mapplanner

import (
	"maps"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestRowFacings_並びごとに背が塞がり前が空く側を向く(t *testing.T) {
	t.Parallel()

	type coord = consts.Coord[consts.Tile]
	set := func(cs ...coord) map[coord]bool {
		m := map[coord]bool{}
		for _, c := range cs {
			m[c] = true
		}
		return m
	}
	weightOf := func(blocked map[coord]bool) func(coord) float64 {
		return func(c coord) float64 {
			if blocked[c] {
				return 1
			}
			return 0
		}
	}
	all := func(f gc.Facing, cs ...coord) map[coord]gc.Facing {
		m := map[coord]gc.Facing{}
		for _, c := range cs {
			m[c] = f
		}
		return m
	}
	row := []coord{{X: 2, Y: 5}, {X: 3, Y: 5}, {X: 4, Y: 5}}
	column := []coord{{X: 5, Y: 2}, {X: 5, Y: 3}, {X: 5, Y: 4}}
	cases := []struct {
		name    string
		facers  []coord
		blocked map[coord]bool
		want    map[coord]gc.Facing
	}{
		{"両面とも空いた東西の並びは南を向き、真ん中に立つ", row, nil, all(gc.Facing{Dir: consts.TileSouth}, row...)},
		{"北に1つだけ塞がりがある並びも全員が南を向き、背へ寄る", row, set(coord{X: 3, Y: 4}), all(gc.Facing{Dir: consts.TileSouth, Backed: true}, row...)},
		{"東の壁と北の角に接する南北の並びは全員が西を向く", column, set(coord{X: 6, Y: 2}, coord{X: 6, Y: 3}, coord{X: 6, Y: 4}, coord{X: 5, Y: 1}), all(gc.Facing{Dir: consts.TileWest, Backed: true}, column...)},
		{"北の壁に接する1台は南を向く", []coord{{X: 5, Y: 5}}, set(coord{X: 5, Y: 4}), all(gc.Facing{Dir: consts.TileSouth, Backed: true}, coord{X: 5, Y: 5})},
		{"何にも接しない1台は南を向き、真ん中に立つ", []coord{{X: 5, Y: 5}}, nil, all(gc.Facing{Dir: consts.TileSouth}, coord{X: 5, Y: 5})},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, RowFacings(set(tc.facers...), weightOf(tc.blocked)), tc.name)
	}

	// 北東の角で、北の壁沿いの5台の列と、角を含めて東の壁沿いに4台の列が角の1台を分け合う。角は長い北の列にそろう
	north5 := []coord{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}, {X: 4, Y: 1}, {X: 5, Y: 1}}
	east3 := []coord{{X: 5, Y: 2}, {X: 5, Y: 3}, {X: 5, Y: 4}}
	walls := set(coord{X: 1, Y: 0}, coord{X: 2, Y: 0}, coord{X: 3, Y: 0}, coord{X: 4, Y: 0}, coord{X: 5, Y: 0},
		coord{X: 6, Y: 1}, coord{X: 6, Y: 2}, coord{X: 6, Y: 3}, coord{X: 6, Y: 4})
	want := all(gc.Facing{Dir: consts.TileSouth, Backed: true}, north5...)
	maps.Copy(want, all(gc.Facing{Dir: consts.TileWest, Backed: true}, east3...))
	facers := set(north5...)
	maps.Copy(facers, set(east3...))
	assert.Equal(t, want, RowFacings(facers, weightOf(walls)), "L字")

	// T字では横の3台の並びが縦の3台の並びと (5,4) を分け合う。同じ長さなら先に見つけた縦の並びが取る
	vertical := []coord{{X: 5, Y: 2}, {X: 5, Y: 3}, {X: 5, Y: 4}}
	horizontal := []coord{{X: 3, Y: 4}, {X: 4, Y: 4}}
	want = all(gc.Facing{Dir: consts.TileSouth}, horizontal...)
	maps.Copy(want, all(gc.Facing{Dir: consts.TileEast}, vertical...))
	facers = set(vertical...)
	maps.Copy(facers, set(horizontal...))
	assert.Equal(t, want, RowFacings(facers, weightOf(nil)), "T字")

	// 2x2 の塊は上段を東西の並びとして先に決め、残りの下段も東西の並びとして背中合わせに南を向く
	block := []coord{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1}}
	want = all(gc.Facing{Dir: consts.TileNorth, Backed: true}, block[:2]...)
	maps.Copy(want, all(gc.Facing{Dir: consts.TileSouth, Backed: true}, block[2:]...))
	assert.Equal(t, want, RowFacings(set(block...), weightOf(set(block...))), "2x2")

	// 北の壁を背にした棚の前に樽があっても、壁を強い根拠にして南を向く
	shelf := coord{X: 5, Y: 5}
	weight := func(c coord) float64 {
		switch c {
		case coord{X: 5, Y: 4}:
			return 2
		case coord{X: 5, Y: 6}:
			return 1
		default:
			return 0
		}
	}
	assert.Equal(t, all(gc.Facing{Dir: consts.TileSouth, Backed: true}, shelf), RowFacings(set(shelf), weight), "壁と樽に挟まれる")
}
