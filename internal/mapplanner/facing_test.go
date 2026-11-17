package mapplanner

import (
	"maps"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestRowPoses_並びごとに背が塞がり前が空く側を向く(t *testing.T) {
	t.Parallel()

	type coord = consts.Coord[consts.Tile]
	set := func(cs ...coord) map[coord]bool {
		m := map[coord]bool{}
		for _, c := range cs {
			m[c] = true
		}
		return m
	}
	noWall := func(coord) bool { return false }
	all := func(f gc.Pose, cs ...coord) map[coord]gc.Pose {
		m := map[coord]gc.Pose{}
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
		want    map[coord]gc.Pose
	}{
		{"両面とも空いた東西の並びは南を向き、真ん中に立つ", row, nil, all(gc.Pose{Facing: consts.South}, row...)},
		{"北に1つだけ塞がりがある並びも全員が南を向き、背へ寄る", row, set(coord{X: 3, Y: 4}), all(gc.Pose{Facing: consts.South, Anchor: gc.AnchorBack}, row...)},
		{"東の壁と北の角に接する南北の並びは全員が西を向く", column, set(coord{X: 6, Y: 2}, coord{X: 6, Y: 3}, coord{X: 6, Y: 4}, coord{X: 5, Y: 1}), all(gc.Pose{Facing: consts.West, Anchor: gc.AnchorBack}, column...)},
		{"北の壁に接する1台は南を向く", []coord{{X: 5, Y: 5}}, set(coord{X: 5, Y: 4}), all(gc.Pose{Facing: consts.South, Anchor: gc.AnchorBack}, coord{X: 5, Y: 5})},
		{"何にも接しない1台は南を向き、真ん中に立つ", []coord{{X: 5, Y: 5}}, nil, all(gc.Pose{Facing: consts.South}, coord{X: 5, Y: 5})},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, RowPoses(set(tc.facers...), tc.blocked, noWall), tc.name)
	}

	// 北東の角で、北の壁沿いの5台の列と、角を含めて東の壁沿いに4台の列が角の1台を分け合う。角は長い北の列にそろう
	north5 := []coord{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1}, {X: 4, Y: 1}, {X: 5, Y: 1}}
	east3 := []coord{{X: 5, Y: 2}, {X: 5, Y: 3}, {X: 5, Y: 4}}
	walls := set(coord{X: 1, Y: 0}, coord{X: 2, Y: 0}, coord{X: 3, Y: 0}, coord{X: 4, Y: 0}, coord{X: 5, Y: 0},
		coord{X: 6, Y: 1}, coord{X: 6, Y: 2}, coord{X: 6, Y: 3}, coord{X: 6, Y: 4})
	want := all(gc.Pose{Facing: consts.South, Anchor: gc.AnchorBack}, north5...)
	maps.Copy(want, all(gc.Pose{Facing: consts.West, Anchor: gc.AnchorBack}, east3...))
	facers := set(north5...)
	maps.Copy(facers, set(east3...))
	assert.Equal(t, want, RowPoses(facers, nil, func(c coord) bool { return walls[c] }), "L字")

	// T字では、交わりの (5,4) を含めて横に3台の並びと縦に3台の並びが (5,4) を分け合う。同じ長さなら先に見つけた縦の並びが取る
	vertical := []coord{{X: 5, Y: 2}, {X: 5, Y: 3}, {X: 5, Y: 4}}
	horizontal := []coord{{X: 3, Y: 4}, {X: 4, Y: 4}}
	want = all(gc.Pose{Facing: consts.South}, horizontal...)
	maps.Copy(want, all(gc.Pose{Facing: consts.East}, vertical...))
	facers = set(vertical...)
	maps.Copy(facers, set(horizontal...))
	assert.Equal(t, want, RowPoses(facers, nil, noWall), "T字")

	// 2x2 の塊は上段を東西の並びとして先に決め、残りの下段も東西の並びとして背中合わせに南を向く
	block := []coord{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1}}
	want = all(gc.Pose{Facing: consts.North, Anchor: gc.AnchorBack}, block[:2]...)
	maps.Copy(want, all(gc.Pose{Facing: consts.South, Anchor: gc.AnchorBack}, block[2:]...))
	assert.Equal(t, want, RowPoses(set(block...), set(block...), noWall), "2x2")

	// 北の壁を背にした棚の前に樽があっても、壁を強い根拠にして南を向く
	shelf := coord{X: 5, Y: 5}
	assert.Equal(t, all(gc.Pose{Facing: consts.South, Anchor: gc.AnchorBack}, shelf), RowPoses(set(shelf), set(coord{X: 5, Y: 6}), func(c coord) bool { return c == coord{X: 5, Y: 4} }), "壁と樽に挟まれる")
}
