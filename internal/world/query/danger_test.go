package query

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestDangerLevel(t *testing.T) {
	t.Parallel()

	t.Run("0日目は最小の1", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevelForDay(0))
	})

	t.Run("負の日数も1に丸める", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevelForDay(-5))
	})

	t.Run("dangerDaysPerLevel未満はまだ1段目", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevelForDay(dangerDaysPerLevel-1))
	})

	t.Run("dangerDaysPerLevel経過でちょうど1段上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2, DangerLevelForDay(dangerDaysPerLevel))
	})

	t.Run("複数段の経過も比例して上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 4, DangerLevelForDay(dangerDaysPerLevel*3))
	})
}

func TestDangerLevelForDepth(t *testing.T) {
	t.Parallel()

	t.Run("起点は最小の1", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevelForDepth(0))
	})

	t.Run("負の深度も1に丸める", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevelForDepth(-5))
	})

	t.Run("dangerChunksPerLevel未満はまだ1段目", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevelForDepth(dangerChunksPerLevel-1))
	})

	t.Run("dangerChunksPerLevel進むとちょうど1段上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2, DangerLevelForDepth(dangerChunksPerLevel))
	})

	t.Run("複数段の北進も比例して上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 4, DangerLevelForDepth(dangerChunksPerLevel*3))
	})
}

func TestDangerLevelAt_worldのゲーム内時間から危険度を求める(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	assert.Equal(t, 1, DangerLevelAt(world), "初期状態は1日目で危険度1")

	GetGameTime(world).TotalTurns = 3000
	assert.Equal(t, 2, DangerLevelAt(world), "3日目は危険度2")
}
