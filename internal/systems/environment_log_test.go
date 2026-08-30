package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogEnvironmentChange(t *testing.T) {
	t.Parallel()

	t.Run("日の入りのターンは日が沈んだログを出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = 250

		logEnvironmentChange(world)

		hist := query.GetGameLog(world).GetHistory()
		require.Len(t, hist, 1)
		assert.Contains(t, hist[0], "The sun sets.")
	})

	t.Run("日の出のターンは日が昇ったログを出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = 1000

		logEnvironmentChange(world)

		hist := query.GetGameLog(world).GetHistory()
		require.Len(t, hist, 1)
		assert.Contains(t, hist[0], "The sun rises.")
	})

	t.Run("季節の変わるターンは季節と日の出の両方を出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// 春から夏へ切り替わる経過ターン。日の始まりは夜明けなので日の出とも重なる
		query.GetGameTime(world).TotalTurns = 11500

		logEnvironmentChange(world)

		hist := query.GetGameLog(world).GetHistory()
		require.Len(t, hist, 2)
		assert.Contains(t, hist[0], "The season changed to")
		assert.Contains(t, hist[0], "Summer")
		assert.Contains(t, hist[1], "The sun rises.")
	})

	t.Run("時間帯も季節も変わらないターンはログを出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns = 1001

		logEnvironmentChange(world)

		assert.Equal(t, 0, query.GetGameLog(world).Count())
	})
}
