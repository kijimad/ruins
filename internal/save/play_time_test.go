package save

import (
	"testing"
	"time"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayTime_保存とロードで往復する(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	want := 101*time.Hour + 34*time.Minute
	query.GetPlayTime(world).Duration = want
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	// 封筒の部分読みで取り出せる
	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err)
	assert.Equal(t, want, got, "封筒から累積プレイ時間を読める")

	// ロードで world の singleton に復元され継続できる
	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))
	assert.Equal(t, want, query.GetPlayTime(fresh).Duration, "ロードで累積プレイ時間が復元される")
}

func TestPlayTime_セーブ時にセッション経過を加算する(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	// 既存の累積 2h に、1時間前に始めたセッションの経過を畳む
	query.GetPlayTime(world).Duration = 2 * time.Hour
	world.Resources.PlayTimeSessionStart = time.Now().Add(-time.Hour)
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err)
	assert.Greater(t, got, 2*time.Hour+50*time.Minute, "セーブ時にセッション経過を足す")
	assert.Less(t, got, 2*time.Hour+70*time.Minute)
	assert.Equal(t, got, query.GetPlayTime(world).Duration, "world 側の PlayTime も畳まれる")
}

func TestPlayTime_基準ゼロのラン外セーブは加算しない(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t) // PlayTimeSessionStart はゼロ
	query.GetPlayTime(world).Duration = 5 * time.Minute
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, got, "基準ゼロでは加算しない")
}

func TestGetSavePlayTime_ゼロは正常値(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t) // PlayTime は 0
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err, "0 はエラーでなく正常値")
	assert.Zero(t, got)
}
