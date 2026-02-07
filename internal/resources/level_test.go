package resources

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestStateChange(t *testing.T) {
	t.Parallel()

	t.Run("正常に状態変更を要求できる", func(t *testing.T) {
		t.Parallel()
		dungeon := &Dungeon{}

		err := dungeon.RequestStateChange(WarpNextEvent{
			NextPlannerType: consts.PlannerTypeNameSmallRoom,
		})
		require.NoError(t, err)

		event := dungeon.ConsumeStateChange()
		assert.Equal(t, StateEventTypeWarpNext, event.Type())
	})

	t.Run("既にイベントが設定されている場合はエラーを返す", func(t *testing.T) {
		t.Parallel()
		dungeon := &Dungeon{}

		// 最初の状態変更要求は成功
		err := dungeon.RequestStateChange(WarpNextEvent{
			NextPlannerType: consts.PlannerTypeNameSmallRoom,
		})
		require.NoError(t, err)

		// 2回目の状態変更要求はエラー
		err = dungeon.RequestStateChange(WarpPlazaEvent{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "イベントがすでに設定されています")
		assert.Contains(t, err.Error(), string(StateEventTypeWarpNext))
		assert.Contains(t, err.Error(), string(StateEventTypeWarpPlaza))
	})

	t.Run("NoneEventは上書き可能", func(t *testing.T) {
		t.Parallel()
		dungeon := &Dungeon{}

		// NoneEventを設定
		err := dungeon.RequestStateChange(NoneEvent{})
		require.NoError(t, err)

		// NoneEvent後は別のイベントを設定可能
		err = dungeon.RequestStateChange(GameClearEvent{})
		assert.NoError(t, err)

		event := dungeon.ConsumeStateChange()
		assert.Equal(t, StateEventTypeGameClear, event.Type())
	})

	t.Run("ConsumeStateChangeで消費後は新しいイベントを設定可能", func(t *testing.T) {
		t.Parallel()
		dungeon := &Dungeon{}

		// 状態変更要求
		err := dungeon.RequestStateChange(WarpNextEvent{
			NextPlannerType: consts.PlannerTypeNameSmallRoom,
		})
		require.NoError(t, err)

		// イベント消費
		event := dungeon.ConsumeStateChange()
		assert.Equal(t, StateEventTypeWarpNext, event.Type())

		// 消費後は新しいイベントを設定可能
		err = dungeon.RequestStateChange(WarpPlazaEvent{})
		assert.NoError(t, err)

		event = dungeon.ConsumeStateChange()
		assert.Equal(t, StateEventTypeWarpPlaza, event.Type())
	})
}
