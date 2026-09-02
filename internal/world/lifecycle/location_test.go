package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// putOwnerBackpackStack は owner のバックパックへ id を count 個置き、代表を返す。
func putOwnerBackpackStack(world w.World, owner ecs.Entity, id string, count int) ecs.Entity {
	var rep ecs.Entity
	for range count {
		e, err := spawnItemBase(world, id)
		if err != nil {
			panic(err)
		}
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: owner})
		rep = e
	}
	return rep
}

// ownerBackpackCount は owner のバックパックにある scrap_iron の個数を数える。
func ownerBackpackCount(world w.World, owner ecs.Entity) int {
	n := 0
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInBackpack.Get(e).Owner == owner && world.Components.RawID.Get(e).ID == "scrap_iron" {
			n++
		}
	}
	return n
}

// TestMoveToField_所有者からの移送で現ステージへ束縛する は、背包などからフィールドへ置いた物が
// 即座に現ステージへ束縛され、総重量へ乗ることを検証する。次の swap を待つ遅延束縛では、内部で
// 置いた物が退場するまで総重量に現れない不具合の回帰。
func TestMoveToField_所有者からの移送で現ステージへ束縛する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	interior := gc.NewCubeInteriorStage()
	query.SetDungeon(world, &gc.Dungeon{CurrentStage: interior})

	owner := world.ECS.NewEntity()
	_, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

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

	t.Run("countが0以下ならスタック全体をrecipientのバックパックへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()
		item := putOwnerBackpackStack(world, owner, "scrap_iron", 5)

		err := TransferUnits(world, item, recipient, 0)
		require.NoError(t, err)

		assert.Equal(t, 0, ownerBackpackCount(world, owner), "owner 側は空になる")
		assert.Equal(t, 5, ownerBackpackCount(world, recipient), "recipient 側へ全部移る")
	})

	t.Run("countが所持数以上ならスタック全体をrecipientのバックパックへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()
		item := putOwnerBackpackStack(world, owner, "scrap_iron", 5)

		err := TransferUnits(world, item, recipient, 10)
		require.NoError(t, err)

		assert.Equal(t, 0, ownerBackpackCount(world, owner))
		assert.Equal(t, 5, ownerBackpackCount(world, recipient))
	})

	t.Run("countが所持数未満なら指定数だけrecipientへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()
		item := putOwnerBackpackStack(world, owner, "scrap_iron", 5)

		err := TransferUnits(world, item, recipient, 2)
		require.NoError(t, err)

		// 1個1エンティティなので、同一スタックのうち2個だけ recipient へ移り、残り3個は owner に残る
		assert.Equal(t, 3, ownerBackpackCount(world, owner), "owner に残る")
		assert.Equal(t, 2, ownerBackpackCount(world, recipient), "recipient へ移る")
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
		item, err := spawnItemBase(world, "wooden_sword")
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

	t.Run("他の所有者の装備は影響を受けない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)
		// プレイヤーは1体という不変条件があるため、別の所有者は素のエンティティで表す。
		// UnequipAll の関心は所有者の分離で、所有者がプレイヤーである必要はない
		other := world.ECS.NewEntity()

		item, err := SpawnBackpackItem(world, "claymore", 1)
		require.NoError(t, err)
		MoveToEquip(world, item, other, gc.SlotWeapon1)

		// プレイヤーの装備解除
		err = UnequipAll(world, player)
		require.NoError(t, err)

		// 別所有者の装備は残っている
		assert.True(t, world.Components.LocationEquipped.Has(item))
	})
}

func TestRemoveOwnedStorage(t *testing.T) {
	t.Parallel()

	t.Run("owner指定なしでは何も削除しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		storage := world.ECS.NewEntity()
		item, err := SpawnStorageItem(world, "wooden_sword", 1, storage)
		require.NoError(t, err)

		RemoveOwnedStorage(world, nil)

		assert.True(t, world.ECS.Alive(item), "owner未指定では在庫は残る")
	})

	t.Run("指定した所有者の収納在庫を削除する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		storage := world.ECS.NewEntity()
		item, err := SpawnStorageItem(world, "wooden_sword", 1, storage)
		require.NoError(t, err)

		RemoveOwnedStorage(world, []ecs.Entity{storage})

		assert.False(t, world.ECS.Alive(item), "指定した所有者の在庫は削除される")
	})

	t.Run("指定していない所有者の収納在庫は残る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		storageA := world.ECS.NewEntity()
		storageB := world.ECS.NewEntity()
		itemA, err := SpawnStorageItem(world, "wooden_sword", 1, storageA)
		require.NoError(t, err)
		itemB, err := SpawnStorageItem(world, "wooden_sword", 1, storageB)
		require.NoError(t, err)

		RemoveOwnedStorage(world, []ecs.Entity{storageA})

		assert.False(t, world.ECS.Alive(itemA), "指定した所有者Aの在庫は削除される")
		assert.True(t, world.ECS.Alive(itemB), "指定していない所有者Bの在庫は残る")
	})
}
