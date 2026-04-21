package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDoorActivity(t *testing.T) {
	t.Parallel()

	t.Run("閉じた扉を開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 扉を作成（閉じている）
		door := world.Manager.NewEntity()
		door.AddComponent(world.Components.Door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		door.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		door.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		door.AddComponent(world.Components.BlockView, &gc.BlockView{})

		// OpenDoorActivityを実行
		params := ActionParams{
			Actor:  player,
			Target: &door,
		}
		result, err := Execute(&OpenDoorActivity{}, params, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "扉を開くアクションが成功するべき")

		// 扉が開いていることを確認
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.True(t, doorComp.IsOpen, "扉が開いているべき")

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
		params := ActionParams{
			Actor:  player,
			Target: &wall,
		}
		result, err := Execute(&OpenDoorActivity{}, params, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Contains(t, err.Error(), "対象エンティティは扉ではありません")

		world.Manager.DeleteEntity(player)
		world.Manager.DeleteEntity(wall)
	})

	t.Run("ロック済み扉を開こうとするとキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		door := world.Manager.NewEntity()
		door.AddComponent(world.Components.Door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal, Locked: true})
		door.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		door.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		door.AddComponent(world.Components.BlockView, &gc.BlockView{})

		params := ActionParams{
			Actor:  player,
			Target: &door,
		}
		result, err := Execute(&OpenDoorActivity{}, params, world)

		require.NoError(t, err, "ロック済み扉のキャンセルは致命的エラーではない")
		require.NotNil(t, result)
		assert.False(t, result.Success, "ロック済み扉は開けない")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)

		// 扉は閉じたまま
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.False(t, doorComp.IsOpen)
		assert.True(t, doorComp.Locked)

		world.Manager.DeleteEntity(player)
		world.Manager.DeleteEntity(door)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// OpenDoorActivityを実行（Targetなし）
		params := ActionParams{
			Actor: player,
		}
		result, err := Execute(&OpenDoorActivity{}, params, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Contains(t, result.Message, "扉エンティティが指定されていません")

		world.Manager.DeleteEntity(player)
	})
}
