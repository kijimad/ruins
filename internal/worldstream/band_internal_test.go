package worldstream

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

// TestTranslateTileKeyMap は translateTileKeyMap の平行移動・nil map・keep フィルタを固定する。
func TestTranslateTileKeyMap(t *testing.T) {
	t.Parallel()

	t.Run("nil_mapはnilを返す", func(t *testing.T) {
		t.Parallel()

		got := translateTileKeyMap[bool](nil, 1, 2, nil)
		assert.Nil(t, got, "nil src はそのまま nil を返す")
	})

	t.Run("keepがnilなら全キーを平行移動して通す", func(t *testing.T) {
		t.Parallel()

		src := map[gc.GridElement]bool{
			{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}}: true,
			{Coord: consts.Coord[consts.Tile]{X: 5, Y: 9}}: true,
		}

		got := translateTileKeyMap(src, -1, 2, nil)

		assert.Len(t, got, 2, "keep=nil はキーを1つも捨てない")
		assert.True(t, got[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 3}}], "(1,1) は (-1,+2) 平行移動で (0,3) になる")
		assert.True(t, got[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 4, Y: 11}}], "(5,9) は (-1,+2) 平行移動で (4,11) になる")
	})

	t.Run("keepがfalseを返すキーは捨てる", func(t *testing.T) {
		t.Parallel()

		src := map[gc.GridElement]bool{
			{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}}: true,
			{Coord: consts.Coord[consts.Tile]{X: 0, Y: 5}}: true,
		}
		// 平行移動後の Y が 10 未満のキーだけ残す
		keep := func(g gc.GridElement) bool { return g.Y < 10 }

		got := translateTileKeyMap(src, 0, 8, keep)

		assert.Len(t, got, 1, "keep が false を返すキーは1つ捨てられる")
		assert.True(t, got[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 8}}], "(0,0)→(0,8) はkeepでtrueなので残る")
		assert.NotContains(t, got, gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 13}}, "(0,5)→(0,13) はkeepでfalseなので残らない")
	})
}
