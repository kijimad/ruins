package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/aiinput"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAISystem(t *testing.T) {
	t.Parallel()

	// テスト用のワールド作成
	world := testutil.InitTestWorld(t)

	// プレイヤーエンティティを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(10), Y: consts.Tile(10)})

	// AIエンティティを作成
	aiEntity := world.Manager.NewEntity()
	aiEntity.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
	aiEntity.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})
	aiEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(5), Y: consts.Tile(5)})
	aiEntity.AddComponent(world.Components.AIVision, &gc.AIVision{
		ViewDistance: 3, // 3タイルの視界
		TargetEntity: &player,
	})
	aiEntity.AddComponent(world.Components.AIState, &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	})
	aiEntity.AddComponent(world.Components.AIPolicy, &gc.AIPolicy{
		Planner:       gc.PlannerRoaming,
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.MovementRandom,
	})

	// システム実行前の位置を記録
	initialGrid := world.Components.GridElement.Get(aiEntity).(*gc.GridElement)
	initialX, initialY := int(initialGrid.X), int(initialGrid.Y)

	// AIシステムを実行（aiinputパッケージを使用）
	processor := aiinput.NewProcessor()
	require.NoError(t, processor.ProcessNonSquadAI(world))

	// システム実行後の位置を記録
	finalGrid := world.Components.GridElement.Get(aiEntity).(*gc.GridElement)
	finalX, finalY := int(finalGrid.X), int(finalGrid.Y)

	// 位置が変わったかどうかを確認（ランダムな動きなので移動有無は不確定）
	moved := (initialX != finalX) || (initialY != finalY)
	t.Logf("AI移動: (%d,%d) -> (%d,%d), moved: %t", initialX, initialY, finalX, finalY, moved)

	// 状態が適切に管理されているかチェック
	aiState := world.Components.AIState.Get(aiEntity).(*gc.AIState)
	validStates := []gc.AIStateSubState{gc.AIStateWaiting, gc.AIStateDriving, gc.AIStateChasing}
	assert.Contains(t, validStates, aiState.SubState, "AI状態が無効: %v", aiState.SubState)
}
