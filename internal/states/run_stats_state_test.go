package states

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
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

	items := runStatsItems(world)

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
