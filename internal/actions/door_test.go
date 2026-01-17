package actions

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDoorActivity(t *testing.T) {
	t.Parallel()

	t.Run("閉じたドアを開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// ドアを作成（閉じている）
		door := world.Manager.NewEntity()
		door.AddComponent(world.Components.Door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		door.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		door.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		door.AddComponent(world.Components.BlockView, &gc.BlockView{})

		// OpenDoorActivityを実行
		manager := NewActivityManager(logger.New(logger.CategoryAction))
		params := ActionParams{
			Actor:  player,
			Target: &door,
		}
		result, err := manager.Execute(&OpenDoorActivity{}, params, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "ドアを開くアクションが成功するべき")

		// ドアが開いていることを確認
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.True(t, doorComp.IsOpen, "ドアが開いているべき")

		// BlockPassとBlockViewが削除されていることを確認
		assert.False(t, door.HasComponent(world.Components.BlockPass), "BlockPassが削除されているべき")
		assert.False(t, door.HasComponent(world.Components.BlockView), "BlockViewが削除されているべき")

		world.Manager.DeleteEntity(player)
		world.Manager.DeleteEntity(door)
	})

	t.Run("Doorコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// 普通の壁を作成（Doorコンポーネントなし）
		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})

		// OpenDoorActivityを実行
		manager := NewActivityManager(logger.New(logger.CategoryAction))
		params := ActionParams{
			Actor:  player,
			Target: &wall,
		}
		result, err := manager.Execute(&OpenDoorActivity{}, params, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Contains(t, err.Error(), "対象エンティティはドアではありません")

		world.Manager.DeleteEntity(player)
		world.Manager.DeleteEntity(wall)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// OpenDoorActivityを実行（Targetなし）
		manager := NewActivityManager(logger.New(logger.CategoryAction))
		params := ActionParams{
			Actor: player,
		}
		result, err := manager.Execute(&OpenDoorActivity{}, params, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Contains(t, result.Message, "ドアエンティティが指定されていません")

		world.Manager.DeleteEntity(player)
	})
}
