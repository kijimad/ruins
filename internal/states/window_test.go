package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateFocusIndex(t *testing.T) {
	t.Parallel()

	t.Run("ActionWindowUp で前の項目に移動", func(t *testing.T) {
		t.Parallel()

		focusIndex := 2
		itemCount := 5

		result := UpdateFocusIndex(inputmapper.ActionWindowUp, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 1, focusIndex, "フォーカスインデックスが1減るべき")
	})

	t.Run("ActionWindowUp で最初の項目から最後の項目にループ", func(t *testing.T) {
		t.Parallel()

		focusIndex := 0
		itemCount := 5

		result := UpdateFocusIndex(inputmapper.ActionWindowUp, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 4, focusIndex, "フォーカスインデックスが最後の項目にループするべき")
	})

	t.Run("ActionWindowDown で次の項目に移動", func(t *testing.T) {
		t.Parallel()

		focusIndex := 2
		itemCount := 5

		result := UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 3, focusIndex, "フォーカスインデックスが1増えるべき")
	})

	t.Run("ActionWindowDown で最後の項目から最初の項目にループ", func(t *testing.T) {
		t.Parallel()

		focusIndex := 4
		itemCount := 5

		result := UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 0, focusIndex, "フォーカスインデックスが最初の項目にループするべき")
	})

	t.Run("項目が1個の場合のActionWindowUp", func(t *testing.T) {
		t.Parallel()

		focusIndex := 0
		itemCount := 1

		result := UpdateFocusIndex(inputmapper.ActionWindowUp, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 0, focusIndex, "フォーカスインデックスが0のままであるべき")
	})

	t.Run("項目が1個の場合のActionWindowDown", func(t *testing.T) {
		t.Parallel()

		focusIndex := 0
		itemCount := 1

		result := UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 0, focusIndex, "フォーカスインデックスが0のままであるべき")
	})

	t.Run("ウィンドウアクション以外は更新されない", func(t *testing.T) {
		t.Parallel()

		focusIndex := 2
		itemCount := 5

		result := UpdateFocusIndex(inputmapper.ActionWindowConfirm, &focusIndex, itemCount)

		assert.False(t, result, "更新が失敗するべき")
		assert.Equal(t, 2, focusIndex, "フォーカスインデックスが変更されないべき")
	})

	t.Run("項目数が0の場合のActionWindowDown", func(t *testing.T) {
		t.Parallel()

		focusIndex := 0
		itemCount := 0

		result := UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)

		require.True(t, result, "更新が成功するべき")
		assert.Equal(t, 0, focusIndex, "フォーカスインデックスが0のままであるべき")
	})

	t.Run("複数回のナビゲーション", func(t *testing.T) {
		t.Parallel()

		focusIndex := 0
		itemCount := 3

		// Down -> Down -> Down (ループして0に戻る)
		UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)
		assert.Equal(t, 1, focusIndex)
		UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)
		assert.Equal(t, 2, focusIndex)
		UpdateFocusIndex(inputmapper.ActionWindowDown, &focusIndex, itemCount)
		assert.Equal(t, 0, focusIndex, "3回Downで0に戻るべき")

		// Up -> Up -> Up (ループして0に戻る)
		UpdateFocusIndex(inputmapper.ActionWindowUp, &focusIndex, itemCount)
		assert.Equal(t, 2, focusIndex)
		UpdateFocusIndex(inputmapper.ActionWindowUp, &focusIndex, itemCount)
		assert.Equal(t, 1, focusIndex)
		UpdateFocusIndex(inputmapper.ActionWindowUp, &focusIndex, itemCount)
		assert.Equal(t, 0, focusIndex, "3回Upで0に戻るべき")
	})
}
