package states

import (
	"strings"
	"testing"
	"time"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunStatsItems は統計テーブルの各行へ RunStats と GameTime の値が反映されることを確認する。
// 結果画面と道中の統計画面はこの行を共用するので、行の組み立てを1点で検証する
func TestRunStatsItems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	s := query.GetRunStats(world)
	require.NotNil(t, s)
	s.EnemiesKilled = 42
	s.ItemsScavenged = 13
	s.SalesTotal = 999
	// TotalTurns は run 開始からの経過ターンそのもの。表示はこの値をそのまま出す
	query.GetGameTime(world).TotalTurns = 5678

	items := runStatsItems(world, 0)

	// ラベルの訳に依存しないよう、各統計値が行の値に現れることで検証する
	values := make([]string, len(items))
	for i, it := range items {
		values[i] = it.Value
	}
	joined := strings.Join(values, ",")
	for _, want := range []string{"5678", "42", "13", "999"} {
		assert.Contains(t, joined, want, "統計行に値 %s が含まれる", want)
	}
}

func TestRunStatsState_結果画面は開いた時点のプレイ時間を出し続ける(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state es.StateFactory[w.World]
		want  time.Duration
	}{
		{"死の結果画面は止める", NewRunResultState, 10 * time.Second},
		{"クリアの画面は止める", NewRunClearState, 10 * time.Second},
		{"道中の統計画面は進める", NewRunStatsState, 15 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			start := time.Unix(0, 0)
			query.GetPlayTime(world).Start(10*time.Second, start)

			state, err := tt.state()
			require.NoError(t, err)
			st, ok := state.(*RunStatsState)
			require.True(t, ok)
			require.NoError(t, st.OnStart(world))
			query.GetPlayTime(world).Tick(start.Add(5 * time.Second))

			props, err := st.Fetch(world)
			require.NoError(t, err)
			assert.Contains(t, props.Tabs[0].Items, statusItemData{Label: query.T(world, "Play time"), Value: formatPlayTime(tt.want)})
		})
	}
}
