package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSleepConditions_CanSleep(t *testing.T) {
	t.Parallel()

	// ブロック条件は疲労・気温・安全。寝具は妨げない
	base := SleepConditions{
		HasFatigue:    true,
		Fatigue:       gc.FatigueExhausted,
		HasAmbient:    true,
		TemperatureOK: true,
		AreaSafe:      true,
	}

	t.Run("疲労し適温で安全なら眠れる", func(t *testing.T) {
		t.Parallel()
		assert.False(t, base.TooTired())
		assert.True(t, base.CanSleep())
	})

	t.Run("快調では眠れない", func(t *testing.T) {
		t.Parallel()
		sc := base
		sc.Fatigue = gc.FatigueRested
		assert.True(t, sc.TooTired())
		assert.False(t, sc.CanSleep())
	})

	t.Run("疲労コンポーネントが無いと眠れない", func(t *testing.T) {
		t.Parallel()
		sc := base
		sc.HasFatigue = false
		assert.True(t, sc.TooTired())
		assert.False(t, sc.CanSleep())
	})

	t.Run("気温が範囲外だと眠れない", func(t *testing.T) {
		t.Parallel()
		sc := base
		sc.TemperatureOK = false
		assert.False(t, sc.CanSleep())
	})

	t.Run("敵が近いと眠れない", func(t *testing.T) {
		t.Parallel()
		sc := base
		sc.AreaSafe = false
		assert.False(t, sc.CanSleep())
	})

	t.Run("普通の疲労でも眠れる", func(t *testing.T) {
		t.Parallel()
		sc := base
		sc.Fatigue = gc.FatigueNormal
		assert.False(t, sc.TooTired())
		assert.True(t, sc.CanSleep())
	})
}

func TestEvaluateSleepConditions_プレイヤーの疲労と安全と寝具を反映する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 過労へ追い込む。Current を上限にして GetLevel が Exhausted を返すようにする
	fatigue := world.Components.Fatigue.Get(player)
	fatigue.Current = fatigue.Max

	sc := EvaluateSleepConditions(world, player)

	assert.True(t, sc.HasFatigue, "プレイヤーは疲労を持つ")
	assert.Equal(t, gc.FatigueExhausted, sc.Fatigue, "過労段階を反映する")
	assert.False(t, sc.TooTired(), "過労は眠れる")
	assert.True(t, sc.AreaSafe, "敵が居なければ安全")
	assert.Equal(t, consts.PercentBase, sc.BeddingQuality, "寝具が無ければ地べたの基準")
}
