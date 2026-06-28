package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMovePlayerToPosition(t *testing.T) {
	t.Parallel()

	t.Run("正常にプレイヤーの位置を更新できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
		player.AddComponent(world.Components.Camera, &gc.Camera{})

		// プレイヤーを移動
		err := MovePlayerToPosition(world, 10, 15)
		require.NoError(t, err)

		// 位置が更新されていることを確認
		gridElement := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, consts.Tile(10), gridElement.X)
		assert.Equal(t, consts.Tile(15), gridElement.Y)
	})

	t.Run("プレイヤーが存在しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで実行
		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})

	t.Run("必須コンポーネントが欠けている場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
		player.AddComponent(world.Components.Camera, &gc.Camera{})

		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})
}

func TestUnequipAll(t *testing.T) {
	t.Parallel()

	t.Run("装備中のアイテムが全てバックパックに移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// 装備アイテムを2つ作成
		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Name, &gc.Name{Name: "武器A"})
		item1.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotWeapon1,
		})

		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Name, &gc.Name{Name: "防具A"})
		item2.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotTorso,
		})

		err := UnequipAll(world, player)
		require.NoError(t, err)

		// 装備が外れている
		assert.False(t, item1.HasComponent(world.Components.LocationEquipped))
		assert.False(t, item2.HasComponent(world.Components.LocationEquipped))

		// バックパックに移動している
		assert.True(t, item1.HasComponent(world.Components.LocationInBackpack))
		assert.True(t, item2.HasComponent(world.Components.LocationInBackpack))
	})

	t.Run("装備なしでもエラーにならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		err := UnequipAll(world, player)
		require.NoError(t, err)
	})

	t.Run("他プレイヤーの装備は影響を受けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player1 := world.Manager.NewEntity()
		player1.AddComponent(world.Components.Player, &gc.Player{})

		player2 := world.Manager.NewEntity()

		// player2の装備
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Name, &gc.Name{Name: "他人の武器"})
		item.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
			Owner:         player2,
			EquipmentSlot: gc.SlotWeapon1,
		})

		// player1の装備解除
		err := UnequipAll(world, player1)
		require.NoError(t, err)

		// player2の装備は残っている
		assert.True(t, item.HasComponent(world.Components.LocationEquipped))
	})
}
