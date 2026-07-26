package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

func TestSpawnSettlement_村は全サービスのNPCが揃う(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	require.NoError(t, spawnSettlement(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, true))
	counts := namesInWorld(world)
	assert.Equal(t, 1, counts["商人"], "交易")
	assert.Equal(t, 1, counts["酒場の主人"], "雇用")
	assert.Equal(t, 1, counts["怪しい科学者"], "合成")
}

func TestSpawnSettlement_一軒家は商人だけの行商拠点(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	require.NoError(t, spawnSettlement(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, false))
	counts := namesInWorld(world)
	assert.Equal(t, 1, counts["商人"], "行商の商人はいる")
	assert.Zero(t, counts["酒場の主人"], "雇用サービスは村にしかない")
	assert.Zero(t, counts["怪しい科学者"], "合成サービスは村にしかない")
}
