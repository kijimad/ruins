package states_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/save"
	gssys "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 保留のまま保存されたダーティフラグ StatsChanged・WeightDirty がロードで復元され、再計算で松明の
// 灯りが戻ることを固定する。新規開始オートセーブが転写前の保留状態を捉える経路だった。
func TestSaveLoad_保留中のダーティフラグを保存し再導出で松明の灯りが戻る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// SpawnPlayer は松明を初期装備し StatsChanged と WeightDirty を立てる。転写・再計算はまだ走らない
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	require.True(t, world.Components.StatsChanged.Has(player), "転写前は StatsChanged が保留")
	require.True(t, world.Components.WeightDirty.Has(player), "再計算前は WeightDirty が保留")
	require.False(t, world.Components.LightSource.Get(player).Enabled, "転写前の派生光源は消灯")

	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "s"))

	loaded := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(loaded, "s"))
	p2, err := query.GetPlayerEntity(loaded)
	require.NoError(t, err)

	// 保留フラグが保存・復元される
	require.True(t, loaded.Components.StatsChanged.Has(p2), "StatsChanged がロードで復元される")
	require.True(t, loaded.Components.WeightDirty.Has(p2), "WeightDirty がロードで復元される")

	// 次の再計算で松明の灯りが戻る
	require.NoError(t, (&gssys.StatsChangedSystem{}).Update(loaded))
	ls := loaded.Components.LightSource.Get(p2)
	assert.True(t, ls.Enabled, "再導出で松明の灯りが復元される")
	assert.Positive(t, int(ls.Radius), "照明範囲も装備から復元される")
}
