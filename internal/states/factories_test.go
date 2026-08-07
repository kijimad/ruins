package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpeningState(t *testing.T) {
	t.Parallel()

	state, err := NewOpeningState()
	require.NoError(t, err)
	require.NotNil(t, state)
	ms, ok := state.(*MessageState)
	require.True(t, ok, "MessageState型である")

	// メッセージは翻訳のため world から OnStart で組む。ここでは build を直接呼んで内容を検証する
	world := testutil.InitTestWorld(t)
	require.NotNil(t, ms.build, "build が設定されている")
	md := ms.build(world)

	// 最初のページにテキストがある
	require.NotNil(t, md)
	assert.NotEmpty(t, md.TextSegmentLines)

	// 最初のページに背景キーが設定されている
	assert.NotEmpty(t, md.BackgroundKey)

	// 後続ページが連結されている
	assert.True(t, md.HasNextMessages(), "後続メッセージが存在する")
}
