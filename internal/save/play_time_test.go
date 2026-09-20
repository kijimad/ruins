package save

import (
	"testing"
	"time"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayTime_基準からの経過を保存しロードで継続する(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	want := 101*time.Hour + 34*time.Minute
	// 基準を want だけ過去に置くと、いまのプレイ実時間は want になる
	query.GetPlayTime(world).SessionStart = time.Now().Add(-want)
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err)
	assert.InDelta(t, want.Seconds(), got.Seconds(), 5, "基準からの経過が保存される")

	// ロードで基準がずれて復元され、time.Since が累積を続ける
	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))
	elapsed := time.Since(query.GetPlayTime(fresh).SessionStart)
	assert.InDelta(t, want.Seconds(), elapsed.Seconds(), 5, "ロードで累積が復元され継続する")
}

func TestGetSavePlayTime_基準ゼロのラン外はゼロ(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t) // SessionStart はゼロ
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err, "0 はエラーでなく正常値")
	assert.Zero(t, got, "基準ゼロのラン外は0")
}
