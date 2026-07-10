package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDoorActivity(t *testing.T) {
	t.Parallel()

	t.Run("閉じた扉を開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// 扉を作成（閉じている）
		door := world.World.NewEntity()
		world.Components.Door.Add(door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		world.Components.GridElement.Add(door, &gc.GridElement{X: 11, Y: 10})
		world.Components.BlockPass.Add(door, &gc.BlockPass{})
		world.Components.BlockView.Add(door, &gc.BlockView{})

		// OpenDoorActivityを実行
		result, err := Execute(&OpenDoorActivity{Target: door}, player, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "扉を開くアクションが成功するべき")

		// 扉が開いていることを確認
		doorComp := world.Components.Door.Get(door)
		assert.True(t, doorComp.IsOpen, "扉が開いているべき")

		// BlockPassとBlockViewが削除されていることを確認
		assert.False(t, world.Components.BlockPass.Has(door), "BlockPassが削除されているべき")
		assert.False(t, world.Components.BlockView.Has(door), "BlockViewが削除されているべき")

		world.World.RemoveEntity(player)
		world.World.RemoveEntity(door)
	})

	t.Run("Doorコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// 普通の壁を作成（Doorコンポーネントなし）
		wall := world.World.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{X: 11, Y: 10})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})

		// OpenDoorActivityを実行
		result, err := Execute(&OpenDoorActivity{Target: wall}, player, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Contains(t, err.Error(), "対象エンティティは扉ではありません")

		world.World.RemoveEntity(player)
		world.World.RemoveEntity(wall)
	})

	t.Run("ロック済み扉を開こうとするとキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 10, Y: 10})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		door := world.World.NewEntity()
		world.Components.Door.Add(door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal, Locked: true})
		world.Components.GridElement.Add(door, &gc.GridElement{X: 11, Y: 10})
		world.Components.BlockPass.Add(door, &gc.BlockPass{})
		world.Components.BlockView.Add(door, &gc.BlockView{})

		result, err := Execute(&OpenDoorActivity{Target: door}, player, world)

		require.NoError(t, err, "ロック済み扉のキャンセルは致命的エラーではない")
		require.NotNil(t, result)
		assert.False(t, result.Success, "ロック済み扉は開けない")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)

		// 扉は閉じたまま
		doorComp := world.Components.Door.Get(door)
		assert.False(t, doorComp.IsOpen)
		assert.True(t, doorComp.Locked)

		// ロック済みログが出力されていることを確認する
		store := query.GetGameLog(world)
		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "扉はロックされている")

		world.World.RemoveEntity(player)
		world.World.RemoveEntity(door)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// OpenDoorActivityを実行（Targetなし → ゼロ値Entityは扉ではない）
		result, err := Execute(&OpenDoorActivity{}, player, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Contains(t, result.Message, "対象エンティティは扉ではありません")

		world.World.RemoveEntity(player)
	})
}
