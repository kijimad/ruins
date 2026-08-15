package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMoveToField_所有者からの移送で現ステージへ束縛する は、背包などからフィールドへ置いた物が
// 即座に現ステージへ束縛され、総重量へ乗ることを検証する。次の swap を待つ遅延束縛では、内部で
// 置いた物が退場するまで総重量に現れない不具合の回帰。
func TestMoveToField_所有者からの移送で現ステージへ束縛する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	interior := gc.NewCubeInteriorStage()
	query.SetDungeon(world, &gc.Dungeon{CurrentStage: interior})

	owner := world.ECS.NewEntity()

	item, err := SpawnBackpackItem(world, "iron", 1)
	require.NoError(t, err)
	itemWeight := world.Components.Weight.Get(item).Milligram

	MoveToField(world, item, &owner)

	require.True(t, world.Components.StageBound.Has(item), "床へ移すと現ステージへ束縛される")
	assert.Equal(t, interior, world.Components.StageBound.Get(item).Key, "束縛先は現ステージ")
	assert.Equal(t, itemWeight, query.CubeWeight(world, interior), "置いた物が即座に総重量へ乗る")
}

func TestMovePlayerToPosition(t *testing.T) {
	t.Parallel()

	t.Run("正常にプレイヤーの位置を更新できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		// プレイヤーを移動
		err = MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 10, Y: 15})
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
		err := MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 10, Y: 15})
		require.Error(t, err)
		assert.ErrorContains(t, err, "no player entity with required components found")
	})

	t.Run("必須コンポーネントが欠けている場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 必須コンポーネントの欠落を作るため GridElement を外す
		player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		ensureRemoved(world.Components.GridElement, player)

		err = MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 10, Y: 15})
		require.Error(t, err)
		assert.ErrorContains(t, err, "no player entity with required components found")
	})
}

func TestTransferUnits(t *testing.T) {
	t.Parallel()

	t.Run("countが0以下ならitem全体をrecipientのバックパックへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()

		item, err := spawnItemBase(world, "scrap_iron", 5)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		err = TransferUnits(world, item, recipient, 0)
		require.NoError(t, err)

		require.True(t, world.Components.LocationInBackpack.Has(item), "移動先はバックパック")
		assert.Equal(t, recipient, world.Components.LocationInBackpack.Get(item).Owner)
		assert.Equal(t, 5, world.Components.Stackable.Get(item).Count, "個数はそのまま")
	})

	t.Run("countが所持数以上ならitem全体をrecipientのバックパックへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()

		item, err := spawnItemBase(world, "scrap_iron", 5)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		err = TransferUnits(world, item, recipient, 10)
		require.NoError(t, err)

		require.True(t, world.Components.LocationInBackpack.Has(item))
		assert.Equal(t, recipient, world.Components.LocationInBackpack.Get(item).Owner)
		assert.Equal(t, 5, world.Components.Stackable.Get(item).Count, "個数はそのまま")
	})

	t.Run("countが所持数未満なら指定数だけ切り出してrecipientへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()

		item, err := spawnItemBase(world, "scrap_iron", 5)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		err = TransferUnits(world, item, recipient, 2)
		require.NoError(t, err)

		// 元アイテムはowner側に残り、指定数だけ減っている
		require.True(t, world.Components.LocationInBackpack.Has(item))
		assert.Equal(t, owner, world.Components.LocationInBackpack.Get(item).Owner)
		assert.Equal(t, 3, world.Components.Stackable.Get(item).Count, "元スタックはcount分減る")

		// recipient側に切り出した新規アイテムが1つ生成されている
		var found ecs.Entity
		var matched int
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.RawID](world.ECS).Query()
		for q.Next() {
			e := q.Entity()
			if e == item {
				continue
			}
			if world.Components.LocationInBackpack.Get(e).Owner == recipient {
				found = e
				matched++
			}
		}
		require.Equal(t, 1, matched, "recipient宛てのアイテムが1つ生成される")
		assert.Equal(t, 2, world.Components.Stackable.Get(found).Count, "切り出した個数はcountぶん")
		assert.Equal(t, "scrap_iron", world.Components.RawID.Get(found).ID)
	})
}

func TestMoveToEquip(t *testing.T) {
	t.Parallel()

	t.Run("フィールドから装備する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.ECS.NewEntity()

		// 移動前はフィールドに置かれ、座標を持つ状態にする
		item, err := SpawnFieldItem(world, "wooden_sword", 3, 4, 1)
		require.NoError(t, err)

		MoveToEquip(world, item, owner, gc.SlotWeapon1)

		require.True(t, world.Components.LocationEquipped.Has(item), "装備状態になる")
		loc := world.Components.LocationEquipped.Get(item)
		assert.Equal(t, owner, loc.Owner)
		assert.Equal(t, gc.SlotWeapon1, loc.EquipmentSlot)

		assert.False(t, world.Components.LocationOnField.Has(item), "フィールドの位置情報は排他的に外れる")
		assert.False(t, world.Components.GridElement.Has(item), "装備するとGridElementは外れる")

		assert.True(t, world.Components.StatsChanged.Has(owner), "所有者にStatsChangedが付与される")
		assert.True(t, world.Components.WeightDirty.Has(owner), "所有者にWeightDirtyが付与される")
	})

	t.Run("バックパックから装備すると元オーナーと新オーナーの両方にWeightDirtyが付く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		prevOwner := world.ECS.NewEntity()
		newOwner := world.ECS.NewEntity()

		// 移動前は元オーナーのバックパックに入っている状態にする
		item, err := spawnItemBase(world, "wooden_sword", 1)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: prevOwner})

		MoveToEquip(world, item, newOwner, gc.SlotWeapon1)

		require.True(t, world.Components.LocationEquipped.Has(item), "装備状態になる")
		loc := world.Components.LocationEquipped.Get(item)
		assert.Equal(t, newOwner, loc.Owner)

		assert.False(t, world.Components.LocationInBackpack.Has(item), "バックパックの位置情報は排他的に外れる")
		assert.True(t, world.Components.WeightDirty.Has(prevOwner), "元オーナーにWeightDirtyが付与される")
		assert.True(t, world.Components.WeightDirty.Has(newOwner), "新オーナーにWeightDirtyが付与される")
	})
}

func TestUnequipAll(t *testing.T) {
	t.Parallel()

	t.Run("装備中のアイテムが全てバックパックに移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)

		// 実アイテムをスポーンし、実経路で装備する
		item1, err := SpawnBackpackItem(world, "claymore", 1)
		require.NoError(t, err)
		MoveToEquip(world, item1, player, gc.SlotWeapon1)

		item2, err := SpawnBackpackItem(world, "western_armor", 1)
		require.NoError(t, err)
		MoveToEquip(world, item2, player, gc.SlotTorso)

		err = UnequipAll(world, player)
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

		// UnequipAll は装備ゼロのプレイヤーでも安全に動くべきだが、SpawnPlayer は初期装備の
		// 松明を必ず装備するため装備ゼロの状態を作れない。ここは装備が一切ないという作為的状態を
		// 検証する目的なので、素のオーナーエンティティを使う
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		err := UnequipAll(world, player)
		require.NoError(t, err)
	})

	t.Run("他プレイヤーの装備は影響を受けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player1, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)
		player2, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)

		// player2の装備を実経路で用意する
		item, err := SpawnBackpackItem(world, "claymore", 1)
		require.NoError(t, err)
		MoveToEquip(world, item, player2, gc.SlotWeapon1)

		// player1の装備解除
		err = UnequipAll(world, player1)
		require.NoError(t, err)

		// player2の装備は残っている
		assert.True(t, world.Components.LocationEquipped.Has(item))
	})
}
