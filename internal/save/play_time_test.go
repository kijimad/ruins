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
