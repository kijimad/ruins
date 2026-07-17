package mapspawner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpawnAt_オフセット配置 は SpawnAt が全エンティティをオフセット領域へ配置することを固定する。
// シームレスワールドで東スラブへチャンクを置く（§5）ための基盤。
func TestSpawnAt_オフセット配置(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const wdt, hgt consts.Tile = 20, 20
	plan, err := mapplanner.Plan(world, wdt, hgt, 1, mapplanner.PlannerTypeSmallRoom)
	require.NoError(t, err)

	const offX, offY consts.Tile = 100, 50
	if _, err := SpawnAt(world, plan, offX, offY); err != nil {
		require.NoError(t, err)
	}

	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	count := 0
	for query.Next() {
		g := world.Components.GridElement.Get(query.Entity())
		assert.GreaterOrEqual(t, g.X, offX, "X はオフセット以上")
		assert.Less(t, g.X, offX+wdt, "X はオフセット+幅未満")
		assert.GreaterOrEqual(t, g.Y, offY, "Y はオフセット以上")
		assert.Less(t, g.Y, offY+hgt, "Y はオフセット+高さ未満")
		count++
	}
	assert.GreaterOrEqual(t, count, int(wdt*hgt), "少なくとも全タイルぶんのエンティティが配置される")
}

// TestSpawn_オフセットなしは原点配置 は Spawn（=SpawnAt(...,0,0)）が従来どおり原点から配置することを固定する。
func TestSpawn_オフセットなしは原点配置(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const wdt, hgt consts.Tile = 20, 20
	plan, err := mapplanner.Plan(world, wdt, hgt, 1, mapplanner.PlannerTypeSmallRoom)
	require.NoError(t, err)

	if _, err := Spawn(world, plan); err != nil {
		require.NoError(t, err)
	}

	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	minX, minY := consts.Tile(1<<30), consts.Tile(1<<30)
	for query.Next() {
		g := world.Components.GridElement.Get(query.Entity())
		minX = min(minX, g.X)
		minY = min(minY, g.Y)
		assert.Less(t, g.X, wdt, "オフセットなしでは X は幅未満")
		assert.Less(t, g.Y, hgt, "オフセットなしでは Y は高さ未満")
	}
	assert.Equal(t, consts.Tile(0), minX, "原点 X=0 のタイルが存在する")
	assert.Equal(t, consts.Tile(0), minY, "原点 Y=0 のタイルが存在する")
}
