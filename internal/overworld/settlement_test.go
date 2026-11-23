package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// namesInWorld は名前を持つエンティティの名前ごとの数を返す。
func namesInWorld(world w.World) map[string]int {
	counts := map[string]int{}
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		counts[string(world.Components.Name.Get(q.Entity()).Name)]++
	}
	return counts
}

func TestSettlementFeature_集落のチャンクの中央に施設を置く(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	chunkW, chunkH, cols, _ := dungeon.DungeonOverworld.BandShape()
	for _, id := range []string{"village", "hamlet"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			var seed uint64
			var c consts.Coord[consts.Chunk]
		search:
			for s := uint64(1); s < 500; s++ {
				for y := range consts.Chunk(30) {
					for x := range cols {
						cc := consts.Coord[consts.Chunk]{X: x, Y: y}
						if chunkTypeAt(raws, s, cc, cols, 0) == chunkSettlement && settlementAt(raws, s, cc).Id == id {
							seed, c = s, cc
							break search
						}
					}
				}
			}
			require.NotZero(t, seed, "前提: 集落のチャンクが見つかる")

			world := testutil.InitTestWorld(t)
			for y := range chunkH {
				for x := range chunkW {
					_, err := lifecycle.SpawnTile(world, "dirt_tundra", x, y, new(15))
					require.NoError(t, err)
				}
			}
			g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: chunkW, chunkH: chunkH, tiles: &tileIndex{world: world, loX: 0, hiX: chunkW}}
			require.NoError(t, settlementFeature{}.place(world, seed, c, cols, g))

			props := 0
			q := ecs.NewFilter2[gc.GridElement, gc.Prop](world.ECS).Query()
			for q.Next() {
				props++
			}
			assert.Positive(t, props, "施設の家具が置かれる")
		})
	}
}

func TestSettlementFeature_集落でないチャンクには何も置かない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	chunkW, chunkH, cols, _ := dungeon.DungeonOverworld.BandShape()
	c := consts.Coord[consts.Chunk]{Y: 1}
	for chunkTypeAt(raws, 123, c, cols, 0) == chunkSettlement {
		c.Y++
	}
	world := testutil.InitTestWorld(t)
	g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: chunkW, chunkH: chunkH, tiles: &tileIndex{world: world, loX: 0, hiX: chunkW}}
	require.NoError(t, settlementFeature{}.place(world, 123, c, cols, g))
	assert.Empty(t, namesInWorld(world), "集落でないチャンクに施設は出ない")
}

func TestSettlementAt_重みの比で種類を選ぶ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	got := map[string]int{}
	for x := range consts.Chunk(10) {
		for y := range consts.Chunk(10) {
			got[settlementAt(raws, 7, consts.Coord[consts.Chunk]{X: x, Y: y}).Id]++
		}
	}
	want := map[string]int{}
	for x := range consts.Chunk(10) {
		for y := range consts.Chunk(10) {
			id := "hamlet"
			if consts.Mix2D(7^placementOf(testutil.InitTestWorld(t).Resources.RawMaster, rawschema.PointFeatureKindSettlement).Salt, x, y)%10 < 6 {
				id = "village"
			}
			want[id]++
		}
	}
	assert.Equal(t, want, got, "村6と小村4の重みは、ハッシュを10で割った余りが6未満なら村になる")
}

func TestSettlements_全ての施設の絵がチャンクに収まる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	chunkW, chunkH, _, _ := dungeon.DungeonOverworld.BandShape()
	for _, s := range raw.PtrSlice(raws.Settlements) {
		names := loader.FacilityChunkNames(s.Facility)
		require.NotEmpty(t, names, s.Id)
		for _, name := range names {
			_, _, cells, err := loader.LoadTemplateByName(name, 1)
			require.NoError(t, err, name)
			// 絵は回転して置くので、幅と高さの大きい方が短い辺に収まる必要がある
			assert.LessOrEqual(t, max(len(cells), len(cells[0])), int(min(chunkW, chunkH)), name)
			// 集落の施設は扉を持つ。チャンクの生成のテストは原野の散布が置かない扉で施設を見分ける
			doors := 0
			for _, row := range cells {
				for _, cell := range row {
					if cell.Prop == "door" {
						doors++
					}
				}
			}
			assert.Positive(t, doors, name)
		}
	}
}

func TestCenterOrigin(t *testing.T) {
	t.Parallel()

	cellsOf := func(w, h int) [][]maptemplate.MapCell {
		cells := make([][]maptemplate.MapCell, h)
		for y := range cells {
			cells[y] = make([]maptemplate.MapCell, w)
		}
		return cells
	}
	g := chunkGeom{chunkW: 24, chunkH: 24}
	tests := []struct {
		name string
		w, h int
		want consts.Coord[consts.Tile]
	}{
		{name: "偶数の幅は左右に同じだけ空ける", w: 20, h: 20, want: consts.Coord[consts.Tile]{X: 2, Y: 2}},
		{name: "奇数の幅は半分を切り捨てて西と北に1マス多く空ける", w: 13, h: 11, want: consts.Coord[consts.Tile]{X: 6, Y: 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, centerOrigin(g, cellsOf(tt.w, tt.h)))
		})
	}
}
