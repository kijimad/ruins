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
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: consts.Tile(10), Y: consts.Tile(10)})

	// AIエンティティを作成
	aiEntity := world.ECS.NewEntity()
	world.Components.Faction.Add(aiEntity, &gc.Faction{Kind: gc.FactionEnemy})
	world.Components.GridElement.Add(aiEntity, &gc.GridElement{X: consts.Tile(5), Y: consts.Tile(5)})
	world.Components.SoloAI.Add(aiEntity, &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		Movement:              gc.SoloRandom,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
		ViewDistance:          3,
		TargetEntity:          &player,
	})

	// システム実行前の位置を記録
	initialGrid := world.Components.GridElement.Get(aiEntity)
	initialX, initialY := int(initialGrid.X), int(initialGrid.Y)

	// AIシステムを実行（aiinputパッケージを使用）
	processor := aiinput.NewProcessor(world.Config.RNG)
	require.NoError(t, processor.ProcessAll(world))

	// システム実行後の位置を記録
	finalGrid := world.Components.GridElement.Get(aiEntity)
	finalX, finalY := int(finalGrid.X), int(finalGrid.Y)

	// 位置が変わったかどうかを確認（ランダムな動きなので移動有無は不確定）
	moved := (initialX != finalX) || (initialY != finalY)
	t.Logf("AI移動: (%d,%d) -> (%d,%d), moved: %t", initialX, initialY, finalX, finalY, moved)

	// 状態が適切に管理されているかチェック
	aiState := world.Components.SoloAI.Get(aiEntity)
	validStates := []gc.AIStateSubState{gc.AIStateWaiting, gc.AIStateDriving, gc.AIStateChasing}
	assert.Contains(t, validStates, aiState.SubState, "AI状態が無効: %v", aiState.SubState)
}
