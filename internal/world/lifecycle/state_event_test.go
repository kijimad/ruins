package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestStateChange(t *testing.T) {
	t.Parallel()

	t.Run("正常に状態変更を要求できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		err := RequestStateChange(world, gc.WarpNextEvent{})
		require.NoError(t, err)

		req := ConsumeStateChange(world)
		assert.IsType(t, gc.WarpNextEvent{}, req)
	})

	t.Run("既にリクエストが設定されている場合はエラーを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		err := RequestStateChange(world, gc.WarpNextEvent{})
		require.NoError(t, err)

		err = RequestStateChange(world, gc.WarpEscapeEvent{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "リクエストがすでに設定されています")
	})

	t.Run("リクエストがない場合はnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		req := ConsumeStateChange(world)
		assert.Nil(t, req)
	})

	t.Run("消費後は新しいリクエストを設定可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		err := RequestStateChange(world, gc.GameClearEvent{})
		require.NoError(t, err)

		req := ConsumeStateChange(world)
		assert.IsType(t, gc.GameClearEvent{}, req)

		err = RequestStateChange(world, gc.WarpEscapeEvent{})
		require.NoError(t, err)

		req = ConsumeStateChange(world)
		assert.IsType(t, gc.WarpEscapeEvent{}, req)
	})
}
