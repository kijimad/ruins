package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpawnDebugStageFire は hearth の位置に燃焼中の火を置くデバッグ生成を検証する。
func TestSpawnDebugStageFire(t *testing.T) {
	t.Parallel()

	t.Run("hearthがあれば火をその位置に燃焼状態で生成する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		hearthCoord := consts.Coord[consts.Tile]{X: 5, Y: 6}
		h := world.ECS.NewEntity()
		world.Components.RawID.Add(h, &gc.RawID{ID: "hearth"})
		world.Components.GridElement.Add(h, &gc.GridElement{Coord: hearthCoord})

		require.NoError(t, spawnDebugStageFire(world))

		found := false
		q := ecs.NewFilter1[gc.Burning](world.ECS).Query()
		defer q.Close()
		for q.Next() {
			e := q.Entity()
			if world.Components.GridElement.Get(e).Coord == hearthCoord {
				found = true
				assert.Equal(t, consts.Turn(debugStageFireBurnTurns), world.Components.Burning.Get(e).Remaining,
					"デバッグ火は規定ターン数だけ燃える")
			}
		}
		assert.True(t, found, "hearth の位置に燃焼中の火がある")
	})

	t.Run("hearthが無ければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		require.NoError(t, spawnDebugStageFire(world))

		q := ecs.NewFilter1[gc.Burning](world.ECS).Query()
		defer q.Close()
		count := 0
		for q.Next() {
			count++
		}
		assert.Zero(t, count, "火は生成されない")
	})
}

// TestInteractionActionChoices はアクション列を選択肢へ変換する純関数を検証する。
func TestInteractionActionChoices(t *testing.T) {
	t.Parallel()

	t.Run("アクションが無ければ選択肢も空", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, interactionActionChoices(nil))
	})

	t.Run("各アクションをラベル付きの選択肢へ変換する", func(t *testing.T) {
		t.Parallel()
		actions := []InteractionAction{
			{Label: "開ける", Interaction: gc.InteractionDoor},
			{Label: "拾う", Interaction: gc.InteractionItem},
		}
		choices := interactionActionChoices(actions)
		require.Len(t, choices, 2)
		assert.Equal(t, "開ける", choices[0].Label)
		assert.Equal(t, "拾う", choices[1].Label)
	})

	t.Run("Runはプレイヤー不在でエラーを返す", func(t *testing.T) {
		t.Parallel()
		choices := interactionActionChoices([]InteractionAction{{Label: "x", Interaction: gc.InteractionItem}})
		require.Len(t, choices, 1)

		_, err := choices[0].Run(testutil.InitTestWorld(t))
		assert.ErrorContains(t, err, "player")
	})
}
