package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dungeonMenuLabels は選択肢のラベル一覧を返す
func dungeonMenuLabels(world w.World) []string {
	_, choices := dungeonMenuChoices(world)
	labels := make([]string, len(choices))
	for i, c := range choices {
		labels[i] = c.Label
	}
	return labels
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

func TestDungeonMenuChoices_有効時はセーブ項目を出す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// InitTestWorld は開発プロファイルなのでセーブ・ロードが有効
	assert.Contains(t, dungeonMenuLabels(world), "Save game")
}

func TestDungeonMenuChoices_体験版はセーブ項目を出さない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// 体験版はセーブ・ロード無効。プロファイル既定の結果をここでは直接与える
	world.Resources.Config.SaveLoadEnabled = false
	assert.NotContains(t, dungeonMenuLabels(world), "Save game")
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
