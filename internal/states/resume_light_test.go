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

// プレイヤー光源は StatsChangedSystem が装備から都度転写する派生値で、その再計算トリガ StatsChanged は
// ダーティフラグ。転写前、すなわちフラグが保留のまま保存されると派生光源は消灯で焼き付く。フラグを
// 保存対象にしたので、ロードで復元され次の転写で松明の灯りが戻る。WeightDirty も同じ理由で往復する。
// 保存直後に走る新規開始オートセーブがこの保留状態を捉える経路だったため、往復で復元することを固定する。
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

	// 保留フラグが保存・復元される。skipComponents へ戻すとここで落ちる
	require.True(t, loaded.Components.StatsChanged.Has(p2), "StatsChanged がロードで復元される")
	require.True(t, loaded.Components.WeightDirty.Has(p2), "WeightDirty がロードで復元される")

	// 次の再計算で松明の灯りが戻る
	require.NoError(t, (&gssys.StatsChangedSystem{}).Update(loaded))
	ls := loaded.Components.LightSource.Get(p2)
	assert.True(t, ls.Enabled, "再導出で松明の灯りが復元される")
	assert.Positive(t, int(ls.Radius), "照明範囲も装備から復元される")
}
