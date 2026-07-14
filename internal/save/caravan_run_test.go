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
// Beacons は json:"-" で保存せず seed から再構築、動的state（Current/progress/供給）は直列化される。
func TestSaveLoad_CaravanRun(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	world := testutil.InitTestWorld(t)

	// ラン状態を用意し、実際に停留点を辿ってジャンプする（Current/progress を進める）
	run := gc.NewCaravanRun(12345, route.ExpeditionTradeCity)
	for range 2 {
		next := run.Beacons.Outgoing(run.Current)
		require.NotEmpty(t, next)
		run.JumpTo(next[0])
	}
	query.SetCaravanRun(world, run)

	wantSeed := run.Seed
	wantExp := run.Expedition
	wantCurrent := run.Current
	wantProgress := run.CaravanProgress
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
	assert.Equal(t, wantCurrent, loaded.Current)
	assert.Equal(t, wantProgress, loaded.CaravanProgress)
	assert.Equal(t, wantFood, loaded.Supply.Food)

	// Beacons は保存されず、reestablishSingleton で seed から再構築される
	require.NotNil(t, loaded.Beacons, "Beacons が seed から再構築される")
	assert.Equal(t, route.GenerateBeacons(wantExp, wantSeed), loaded.Beacons, "再構築した Beacons は元と一致")
	assert.NotNil(t, loaded.Beacons.BeaconByID(loaded.Current), "復元した Current が有効な停留点を指す")
}
