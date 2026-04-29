package states

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpeningState(t *testing.T) {
	t.Parallel()

	state := NewOpeningState()

	require.NotNil(t, state)
	ms, ok := state.(*MessageState)
	require.True(t, ok, "MessageState型である")

	// メッセージデータが設定されている
	require.NotNil(t, ms.messageData)

	// 最初のページにテキストがある
	assert.NotEmpty(t, ms.messageData.TextSegmentLines)

	// 最初のページに背景キーが設定されている
	assert.NotEmpty(t, ms.messageData.BackgroundKey)

	// 後続ページが連結されている
	assert.True(t, ms.messageData.HasNextMessages(), "後続メッセージが存在する")
}
