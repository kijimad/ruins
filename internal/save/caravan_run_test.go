package save

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/route"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveLoad_CaravanRun はマクロ移動のラン状態が save/load を往復することを検証する。
// Grid は json:"-" で保存せず seed から再構築、動的state（Pos/FrontCol/供給）は直列化される。
func TestSaveLoad_CaravanRun(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	world := testutil.InitTestWorld(t)

	// ラン状態を用意し、実際にグリッドを数セル移動する（Pos/FrontCol を進める）
	run := gc.NewCaravanRun(12345, route.ExpeditionTradeCity)
	for range 2 {
		run.MoveTo(route.Coord{X: run.Pos.X + 1, Y: run.Pos.Y})
	}
	query.SetCaravanRun(world, run)

	wantSeed := run.Seed
	wantExp := run.Expedition
	wantPos := run.Pos
	wantFrontCol := run.FrontCol
	wantFood := run.Supply.Food

	saveManager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)
	require.NoError(t, saveManager.SaveWorld(world, "caravan_slot"))

	// 別ワールドにロード
	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, saveManager.LoadWorld(newWorld, "caravan_slot"))

	loaded := query.GetCaravanRun(newWorld)
	require.NotNil(t, loaded, "CaravanRun が復元される")

	// 動的stateが直列化されて復元される
	assert.Equal(t, wantSeed, loaded.Seed)
	assert.Equal(t, wantExp, loaded.Expedition)
	assert.Equal(t, wantPos, loaded.Pos)
	assert.Equal(t, wantFrontCol, loaded.FrontCol)
	assert.Equal(t, wantFood, loaded.Supply.Food)

	// Grid は保存されず、reestablishSingleton で seed から再構築される
	require.NotNil(t, loaded.Grid, "Grid が seed から再構築される")
	assert.Equal(t, route.GenerateGrid(wantExp, wantSeed, gc.GridW, gc.GridH), loaded.Grid, "再構築した Grid は元と一致")
	assert.True(t, loaded.Grid.In(loaded.Pos), "復元した Pos が再構築グリッド内を指す")
}
