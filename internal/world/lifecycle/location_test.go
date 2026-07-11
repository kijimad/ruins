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
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: 5, Y: 5})
		world.Components.SpriteRender.Add(player, &gc.SpriteRender{})
		world.Components.Camera.Add(player, &gc.Camera{})

		// プレイヤーを移動
		err := MovePlayerToPosition(world, 10, 15)
		require.NoError(t, err)

		// 位置が更新されていることを確認
		gridElement := world.Components.GridElement.Get(player)
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
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.SpriteRender.Add(player, &gc.SpriteRender{})
		world.Components.Camera.Add(player, &gc.Camera{})

		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})
}

func TestMovePlayerToPosition_隊員も隣接位置に再配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, player, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	err = MovePlayerToPosition(world, 20, 20)
	require.NoError(t, err)

	// プレイヤーが移動している
	playerGrid := world.Components.GridElement.Get(player)
	assert.Equal(t, consts.Tile(20), playerGrid.X)
	assert.Equal(t, consts.Tile(20), playerGrid.Y)

	// 隊員がプレイヤーの隣接タイルに配置されている
	memberGrid := world.Components.GridElement.Get(member)
	dx := int(memberGrid.X) - int(playerGrid.X)
	dy := int(memberGrid.Y) - int(playerGrid.Y)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	assert.True(t, dx <= 1 && dy <= 1 && (dx+dy) > 0,
		"隊員はプレイヤーの隣接タイルに配置される: member=(%d,%d) player=(%d,%d)",
		memberGrid.X, memberGrid.Y, playerGrid.X, playerGrid.Y)
}

func TestMovePlayerToPosition_複数隊員が重複しない位置に配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	member1, err := SpawnSquadMember(world, player, "隊員A", testAbilities(), "player")
	require.NoError(t, err)
	member2, err := SpawnSquadMember(world, player, "隊員B", testAbilities(), "player")
	require.NoError(t, err)

	err = MovePlayerToPosition(world, 20, 20)
	require.NoError(t, err)

	m1Grid := world.Components.GridElement.Get(member1)
	m2Grid := world.Components.GridElement.Get(member2)

	// 2人の隊員が異なる位置に配置されている
	assert.False(t, m1Grid.X == m2Grid.X && m1Grid.Y == m2Grid.Y,
		"隊員同士は重複しない位置に配置される")
}

func TestUnequipAll(t *testing.T) {
	t.Parallel()

	t.Run("装備中のアイテムが全てバックパックに移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		// 装備アイテムを2つ作成
		item1 := world.ECS.NewEntity()
		world.Components.Name.Add(item1, &gc.Name{Name: "武器A"})
		world.Components.LocationEquipped.Add(item1, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotWeapon1,
		})

		item2 := world.ECS.NewEntity()
		world.Components.Name.Add(item2, &gc.Name{Name: "防具A"})
		world.Components.LocationEquipped.Add(item2, &gc.LocationEquipped{
			Owner:         player,
			EquipmentSlot: gc.SlotTorso,
		})

		err := UnequipAll(world, player)
		require.NoError(t, err)

		// 装備が外れている
		assert.False(t, world.Components.LocationEquipped.Has(item1))
		assert.False(t, world.Components.LocationEquipped.Has(item2))

		// バックパックに移動している
		assert.True(t, world.Components.LocationInBackpack.Has(item1))
		assert.True(t, world.Components.LocationInBackpack.Has(item2))
	})

	t.Run("装備なしでもエラーにならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		err := UnequipAll(world, player)
		require.NoError(t, err)
	})

	t.Run("他プレイヤーの装備は影響を受けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player1 := world.ECS.NewEntity()
		world.Components.Player.Add(player1, &gc.Player{})

		player2 := world.ECS.NewEntity()

		// player2の装備
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "他人の武器"})
		world.Components.LocationEquipped.Add(item, &gc.LocationEquipped{
			Owner:         player2,
			EquipmentSlot: gc.SlotWeapon1,
		})

		// player1の装備解除
		err := UnequipAll(world, player1)
		require.NoError(t, err)

		// player2の装備は残っている
		assert.True(t, world.Components.LocationEquipped.Has(item))
	})
}
