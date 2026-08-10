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
	item := world.ECS.NewEntity()
	world.Components.Weight.Add(item, &gc.Weight{Milligram: 8 * consts.MilligramPerKg})
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

	MoveToField(world, item, &owner)

	require.True(t, world.Components.StageBound.Has(item), "床へ移すと現ステージへ束縛される")
	assert.Equal(t, interior, world.Components.StageBound.Get(item).Key, "束縛先は現ステージ")
	assert.Equal(t, consts.Milligram(8*consts.MilligramPerKg), query.CubeWeight(world, interior), "置いた物が即座に総重量へ乗る")
}

func TestMovePlayerToPosition(t *testing.T) {
	t.Parallel()

	t.Run("正常にプレイヤーの位置を更新できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		world.Components.SpriteRender.Add(player, &gc.SpriteRender{})
		world.Components.Camera.Add(player, &gc.Camera{})

		// プレイヤーを移動
		err := MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 10, Y: 15})
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

		// GridElementなしのプレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.SpriteRender.Add(player, &gc.SpriteRender{})
		world.Components.Camera.Add(player, &gc.Camera{})

		err := MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 10, Y: 15})
		require.Error(t, err)
		assert.ErrorContains(t, err, "no player entity with required components found")
	})
}

func TestMovePlayerToPosition_隊員も隣接位置に再配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	member, err := SpawnSquadMember(world, player, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	err = MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 20, Y: 20})
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

	player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	member1, err := SpawnSquadMember(world, player, "隊員A", testAbilities(), "player")
	require.NoError(t, err)
	member2, err := SpawnSquadMember(world, player, "隊員B", testAbilities(), "player")
	require.NoError(t, err)

	err = MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: 20, Y: 20})
	require.NoError(t, err)

	m1Grid := world.Components.GridElement.Get(member1)
	m2Grid := world.Components.GridElement.Get(member2)

	// 2人の隊員が異なる位置に配置されている
	assert.False(t, m1Grid.X == m2Grid.X && m1Grid.Y == m2Grid.Y,
		"隊員同士は重複しない位置に配置される")
}

func TestMovePlayerToPosition_地図端の密集でも隊員が重ならず散る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 隊員を多数連れる。近傍リングだけでは全員を収めきれない数にする
	const numMembers = 15
	for range numMembers {
		_, err := SpawnSquadMember(world, player, "隊員", testAbilities(), "player")
		require.NoError(t, err)
	}

	// 地図端 (24,0) の近傍リング(squadPlacementMaxRadius)を壁で埋め、近傍だけでは空きが尽きる
	// 状況を作る。この密集でも全隊員が別タイルへ散ることを検証する。
	target := consts.Coord[consts.Tile]{X: 24, Y: 0}
	for dx := -squadPlacementMaxRadius; dx <= squadPlacementMaxRadius; dx++ {
		for dy := -squadPlacementMaxRadius; dy <= squadPlacementMaxRadius; dy++ {
			x, y := int(target.X)+dx, int(target.Y)+dy
			if x < 0 || y < 0 {
				continue
			}
			wall := world.ECS.NewEntity()
			world.Components.GridElement.Add(wall, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}})
			world.Components.BlockPass.Add(wall, &gc.BlockPass{})
		}
	}
	query.InvalidateSpatialIndex(world)

	require.NoError(t, MovePlayerToPosition(world, target))

	// マップ全体へ探索を広げるので、全隊員が別々のタイルへ散る。1タイルへ重ならない
	seen := map[gc.GridElement]bool{}
	for _, m := range query.SquadMembers(world) {
		g := *world.Components.GridElement.Get(m)
		assert.False(t, seen[g], "隊員が同じタイル (%d,%d) に重なっている", g.X, g.Y)
		seen[g] = true
	}
	assert.Len(t, seen, numMembers, "全隊員が別々のタイルに配置される")
}

// TestSpawnSquadMember_連続生成で重ならない は、隊員を続けて生成しても互いに重ならないことを固定する。
// SpawnSquadMember は末尾で SpatialIndex を無効化するので、次の生成の findPlacementTile が索引を
// 再構築し直前の隊員を占有として見る。ゲーム開始時などの連続生成で同一タイルへ重ならないことの回帰。
func TestSpawnSquadMember_連続生成で重ならない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	const numMembers = 8
	for range numMembers {
		_, err := SpawnSquadMember(world, player, "隊員", testAbilities(), "player")
		require.NoError(t, err)
	}

	seen := map[gc.GridElement]bool{}
	seen[*world.Components.GridElement.Get(player)] = true
	for _, m := range query.SquadMembers(world) {
		g := *world.Components.GridElement.Get(m)
		assert.Falsef(t, seen[g], "(%d,%d) に別キャラと重なった", g.X, g.Y)
		seen[g] = true
	}
	assert.Len(t, seen, numMembers+1, "プレイヤーと全隊員が別タイルに配置される")
}

func TestTransferUnits(t *testing.T) {
	t.Parallel()

	t.Run("countが0以下ならitem全体をrecipientのバックパックへ移す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		owner := world.ECS.NewEntity()
		recipient := world.ECS.NewEntity()

		item := world.ECS.NewEntity()
		world.Components.RawID.Add(item, &gc.RawID{ID: "scrap_iron"})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 5})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		err := TransferUnits(world, item, recipient, 0)
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

		item := world.ECS.NewEntity()
		world.Components.RawID.Add(item, &gc.RawID{ID: "scrap_iron"})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 5})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		err := TransferUnits(world, item, recipient, 10)
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

		item := world.ECS.NewEntity()
		world.Components.RawID.Add(item, &gc.RawID{ID: "scrap_iron"})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 5})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		err := TransferUnits(world, item, recipient, 2)
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
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "テストの剣"})
		// 移動前はフィールドに置かれ、座標を持っている
		world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 4}})

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
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "テストの剣"})
		// 移動前は元オーナーのバックパックに入っている
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
