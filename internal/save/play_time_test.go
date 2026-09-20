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
	// 蓄積を want、今セッションは始まったばかりにすると、いまのプレイ実時間は want になる
	pt := query.GetPlayTime(world)
	pt.Total = want
	pt.SessionStartedAt = time.Now()
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err)
	assert.InDelta(t, want.Seconds(), got.Seconds(), 5, "蓄積＋今セッションの経過が保存される")

	// ロードで蓄積が復元され、以後の time.Since を足して継続する
	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))
	assert.InDelta(t, want.Seconds(), query.PlayTimeElapsed(fresh).Seconds(), 5, "ロードで蓄積が復元され継続する")
}

func TestGetSavePlayTime_基準ゼロのラン外はゼロ(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t) // SessionStartedAt はゼロ
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	got, err := sm.GetSavePlayTime("slot1")
	require.NoError(t, err, "0 はエラーでなく正常値")
	assert.Zero(t, got, "基準ゼロのラン外は0")
}
