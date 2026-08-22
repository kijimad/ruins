package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunResultText は結果画面の本文へ決着と統計の各値が反映されることを確認する
func TestRunResultText(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	s := query.GetRunStats(world)
	require.NotNil(t, s)
	s.EnemiesKilled = 42
	s.ItemsScavenged = 13
	s.SalesTotal = 999
	// 表示は run 開始からの経過ターン。開始原点に 5678 を足せば経過は 5678 になる
	query.GetGameTime(world).TotalTurns = gc.GameStartTurns() + 5678

	text := runResultText(world)

	// ラベルの訳に依存しないよう、各統計値が本文に含まれることで検証する
	for _, want := range []string{"5678", "42", "13", "999"} {
		assert.Contains(t, text, want, "結果テキストに統計値 %s が含まれる", want)
	}
}

// TestRunStatsText は道中の統計画面の本文へ、RunStats と GameTime の各値が反映されることを確認する
func TestRunStatsText(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	s := query.GetRunStats(world)
	require.NotNil(t, s)
	s.EnemiesKilled = 42
	s.ItemsScavenged = 13
	s.SalesTotal = 999
	// 表示は run 開始からの経過ターン。開始原点に 5678 を足せば経過は 5678 になる
	query.GetGameTime(world).TotalTurns = gc.GameStartTurns() + 5678

	text := runStatsText(world)

	for _, want := range []string{"5678", "42", "13", "999"} {
		assert.Contains(t, text, want, "統計テキストに値 %s が含まれる", want)
	}
}

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

func TestNewGameOverMessageState(t *testing.T) {
	t.Parallel()

	state, err := NewGameOverMessageState()
	require.NoError(t, err)
	ms, ok := state.(*MessageState)
	require.True(t, ok, "MessageState型である")

	world := testutil.InitTestWorld(t)
	require.NotNil(t, ms.build, "build が設定されている")
	md := ms.build(world)

	require.NotNil(t, md)
	assert.NotEmpty(t, md.TextSegmentLines, "本文がある")
	assert.NotEmpty(t, md.Choices, "メインメニューへ戻る選択肢がある")
}

func TestNewMerchantDialogState(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()

	state, err := NewMerchantDialogState("商人", merchant)
	require.NoError(t, err)
	ps, ok := state.(*PersistentMessageState)
	require.True(t, ok, "PersistentMessageState型である")

	require.NotNil(t, ps.build, "build が設定されている")
	md := ps.build(world)

	require.NotNil(t, md)
	assert.NotEmpty(t, md.TextSegmentLines, "本文がある")
	assert.Len(t, md.Choices, 2, "見る・取引しないの2択がある")
}
