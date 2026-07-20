package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDescend_現階を退避し訪問済み階を再稼働する は共存方式の下りを検証する。
// 訪問済みの階へ降りると、現階は破棄されず退避され、行き先は再生成でなく再稼働される。
// これが「行き来しても保持」の実挙動。実生成を通らない resume 経路で orchestration を確かめる。
func TestDescend_現階を退避し訪問済み階を再稼働する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 現在は1階
	d := query.GetDungeon(world)
	d.CurrentStage = dungeonStageKey(1)
	d.Depth = 1
	floor1 := addStageEntity(t, world, dungeonStageKey(1))

	// 2階は訪問済みとして退避中に置く。降りると再稼働されるべき
	floor2 := addStageEntity(t, world, dungeonStageKey(2))
	world.Components.Suspended.Add(floor2, &gc.Suspended{})

	st := &DungeonState{Depth: 1}
	require.NoError(t, st.descend(world))

	// 現階は退避され現物が残る。行き先は再稼働される
	assert.True(t, world.Components.Suspended.Has(floor1), "降りた1階は退避される")
	assert.True(t, world.ECS.Alive(floor1), "1階のエンティティは破棄されず現物が残る")
	assert.False(t, world.Components.Suspended.Has(floor2), "再訪する2階は再稼働される")

	// 深度と現ステージが更新される
	assert.Equal(t, 2, st.Depth)
	assert.Equal(t, 2, query.GetDungeon(world).Depth)
	assert.Equal(t, dungeonStageKey(2), query.GetDungeon(world).CurrentStage)
}
