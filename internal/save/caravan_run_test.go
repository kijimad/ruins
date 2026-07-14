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
// Graph は json:"-" で保存せず seed から再構築、動的state（Current/Visited/供給/progress）は直列化される。
func TestSaveLoad_CaravanRun(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	world := testutil.InitTestWorld(t)

	// ラン状態を用意し、実際の辺を辿って現在ノードを進める（Current を有効ノードにする）
	run := gc.NewCaravanRun(12345, route.ExpeditionTradeCity)
	for range 2 {
		out := run.Graph.Outgoing(run.Current)
		require.NotEmpty(t, out)
		run.AdvanceAlong(out[0])
	}
	query.SetCaravanRun(world, run)

	wantSeed := run.Seed
	wantExp := run.Expedition
	wantCurrent := run.Current
	wantVisited := append([]route.NodeID(nil), run.Visited...)
	wantFood := run.Supply.Food
	wantCaravanProgress := run.CaravanProgress
	wantFrontProgress := run.FrontProgress

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
	assert.Equal(t, wantCurrent, loaded.Current)
	assert.Equal(t, wantVisited, loaded.Visited)
	assert.Equal(t, wantFood, loaded.Supply.Food)
	assert.Equal(t, wantCaravanProgress, loaded.CaravanProgress)
	assert.Equal(t, wantFrontProgress, loaded.FrontProgress)

	// Graph は保存されず、reestablishSingleton で seed から再構築される
	require.NotNil(t, loaded.Graph, "Graph が seed から再構築される")
	assert.Equal(t, route.Generate(wantExp, wantSeed), loaded.Graph, "再構築した Graph は元と一致")
	assert.NotNil(t, loaded.Graph.NodeByID(loaded.Current), "復元した Current が再構築グラフの有効ノードを指す")
}
