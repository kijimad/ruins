package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// namesInWorld は名前を持つエンティティの名前ごとの数を返す。
func namesInWorld(world w.World) map[string]int {
	counts := map[string]int{}
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		counts[world.Components.Name.Get(q.Entity()).Name]++
	}
	return counts
}

// settlementRow は raw の settlements の行を id で引く
func settlementRow(t *testing.T, raws oapi.Raws, id string) oapi.Settlement {
	t.Helper()
	for _, s := range raw.PtrSlice(raws.Settlements) {
		if s.Id == id {
			return s
		}
	}
	require.FailNow(t, "settlement not found", id)
	return oapi.Settlement{}
}

func TestSpawnSettlement_村には商人がいる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	require.NoError(t, spawnSettlement(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, settlementRow(t, world.Resources.RawMaster, "village")))
	counts := namesInWorld(world)
	assert.Equal(t, 1, counts["Merchant"], "交易と雇用")
}

func TestSpawnSettlement_一軒家は商人だけの行商拠点(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	require.NoError(t, spawnSettlement(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, settlementRow(t, world.Resources.RawMaster, "hamlet")))
	counts := namesInWorld(world)
	assert.Equal(t, 1, counts["Merchant"], "行商の商人はいる")
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
			if ChunkSeed2D(7^settlementSalt, x, y)%10 < 6 {
				id = "village"
			}
			want[id]++
		}
	}
	assert.Equal(t, want, got, "村6と一軒家4の重みは、ハッシュを10で割った余りが6未満なら村になる")
}
