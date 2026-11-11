package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSleepConditions_CanSleep(t *testing.T) {
	t.Parallel()

	// ブロック条件は疲労・気温・安全。寝具は妨げない
	base := SleepConditions{
		Sleepy:        true,
		TemperatureOK: true,
		AreaSafe:      true,
	}

	t.Run("疲労し適温で安全なら眠れる", func(t *testing.T) {
		t.Parallel()
		assert.False(t, base.TooTired())
		assert.True(t, base.CanSleep())
	})

	t.Run("疲労が眠れる段階に無いと眠れない", func(t *testing.T) {
		t.Parallel()
		sc := base
		sc.Sleepy = false
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
}

func TestEvaluateSleepConditions_疲労と安全を反映する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	// 過労へ追い込む。Current を上限にして GetLevel が Exhausted を返すようにする
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 2000, Max: 2000})

	sc := EvaluateSleepConditions(world, actor)

	assert.True(t, sc.Sleepy, "過労は眠れる段階")
	assert.False(t, sc.TooTired(), "過労は眠れる")
	assert.True(t, sc.AreaSafe, "敵が居なければ安全")
}
