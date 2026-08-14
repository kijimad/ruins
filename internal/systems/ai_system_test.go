package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/aiinput"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAISystem(t *testing.T) {
	t.Parallel()

	// テスト用のワールド作成
	world := testutil.InitTestWorld(t)

	// プレイヤーを実スポーンで配置する
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// AI敵を実スポーンで配置し、プレイヤーを標的にした状態でAI処理を通す
	aiEntity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "moss_turtle")
	require.NoError(t, err)
	solo := world.Components.SoloAI.Get(aiEntity)
	solo.SubState = gc.AIStateWaiting
	solo.ViewDistance = 3
	solo.TargetEntity = &player

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
