package systems

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeadCleanupSystem(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// 1. 通常の敵（AI）エンティティ - 削除されるべき
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 3, Y: 3}, "moss_turtle")
	require.NoError(t, err)
	world.Components.Dead.Add(enemy, &gc.Dead{})

	// 2. プレイヤーエンティティ - 削除されないべき
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	world.Components.Dead.Add(player, &gc.Dead{})

	// 3. その他のDeadエンティティ - 削除されるべき
	// 対応するスポーンの無い最小の Dead 保持エンティティ。Player を持たない Dead が消えることを見る
	otherDead := world.ECS.NewEntity()
	world.Components.Name.Add(otherDead, &gc.Name{Name: "その他"})
	world.Components.Dead.Add(otherDead, &gc.Dead{})

	// 4. 生きているエンティティ - 削除されないべき
	alive, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 7, Y: 7}, "moss_turtle")
	require.NoError(t, err)

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 結果を検証

	// 敵エンティティは削除されているべき（Nameコンポーネントも削除される）
	assert.False(t, world.ECS.Alive(enemy), "敵エンティティは削除されるべき")

	// プレイヤーエンティティは削除されていないべき
	assert.True(t, world.Components.Name.Has(player), "プレイヤーエンティティは削除されないべき")
	assert.True(t, world.Components.Dead.Has(player), "プレイヤーのDeadコンポーネントは残るべき")

	// その他のDeadエンティティは削除されているべき
	assert.False(t, world.ECS.Alive(otherDead), "その他のDeadエンティティは削除されるべき")

	// 生きているエンティティは削除されていないべき
	assert.True(t, world.Components.Name.Has(alive), "生きているエンティティは削除されないべき")
	assert.False(t, world.Components.Dead.Has(alive), "生きているエンティティにDeadコンポーネントはないべき")
}

// TestDeadCleanupSystem_EnemiesKilledStat は敵の除去だけが撃破統計へ加算されることを確認する
func TestDeadCleanupSystem_EnemiesKilledStat(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// 敵の死は撃破としてカウントされる
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 3, Y: 3}, "moss_turtle")
	require.NoError(t, err)
	world.Components.Dead.Add(enemy, &gc.Dead{})

	// プレイヤーの死はカウントしない
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	world.Components.Dead.Add(player, &gc.Dead{})

	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	stats := query.GetRunStats(world)
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.EnemiesKilled, "敵1体の撃破だけがカウントされる")
}

func TestDeadCleanupSystem_NoDeadEntities(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// Deadエンティティが存在しない状態でテスト
	alive1 := world.ECS.NewEntity()
	world.Components.Name.Add(alive1, &gc.Name{Name: "生きている1"})

	alive2 := world.ECS.NewEntity()
	world.Components.Name.Add(alive2, &gc.Name{Name: "生きている2"})
	world.Components.SoloAI.Add(alive2, &gc.SoloAI{})

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// すべてのエンティティが残っているべき
	assert.True(t, world.Components.Name.Has(alive1), "生きているエンティティ1は残るべき")
	assert.True(t, world.Components.Name.Has(alive2), "生きているエンティティ2は残るべき")
}

func TestDeadCleanupSystem_EmptyWorld(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// Deadコンポーネントを持つエンティティがない状態でテスト
	// パニックやエラーが発生しないことを確認する
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// Deadコンポーネントを持つエンティティが存在しないことを確認
	deadCount := 0
	deadQuery := ecs.NewFilter1[gc.Dead](world.ECS).Query()
	for deadQuery.Next() {
		deadCount++
	}
	assert.Equal(t, 0, deadCount, "Deadコンポーネントを持つエンティティは存在しない")
}

func TestDeadCleanupSystem_WithDropTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// ドロップテーブルを持つ敵エンティティを作成（灰の偶像は100%ドロップ）
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "灰の偶像"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.DropTable.Add(enemy, &gc.DropTable{Name: "ash_idol"})
	world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

	// DeadCleanupSystem実行前のアイテムエンティティ数をカウント
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 敵エンティティは削除されているべき
	assert.False(t, world.ECS.Alive(enemy), "敵エンティティは削除されるべき")

	// ドロップアイテムが生成されているべき（"鉄くず"がドロップテーブルに定義されている）
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for itemAfterQuery.Next() {
		itemCountAfter++
	}

	assert.Greater(t, itemCountAfter, itemCountBefore, "ドロップアイテムが生成されているべき")
	assert.Equal(t, itemCountBefore+1, itemCountAfter, "1つのアイテムがドロップされるべき")
}

func TestDeadCleanupSystem_WithDropTableDrops(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// シード2でドロップするケース
	world.Resources.Config.Seed = 2
	world.Resources.Config.RNG = rand.New(rand.NewPCG(world.Resources.Config.Seed, 0))

	// 敵エンティティを作成
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "火の玉"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.DropTable.Add(enemy, &gc.DropTable{Name: "fireball"})
	world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

	// 実行前のアイテム数
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 実行後のアイテム数
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for itemAfterQuery.Next() {
		itemCountAfter++
	}

	// シード2ではドロップする
	assert.Equal(t, itemCountBefore+1, itemCountAfter, "シード2ではドロップするはず")
}

func TestDeadCleanupSystem_WithoutDropTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// ドロップテーブルを持たない敵はアイテムをドロップしない
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "火の玉"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

	// 実行前のアイテム数
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 実行後のアイテム数
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for itemAfterQuery.Next() {
		itemCountAfter++
	}

	assert.Equal(t, itemCountBefore, itemCountAfter, "ドロップテーブルなしではドロップしない")
}

func TestDeadCleanupSystem_CancelsActivity(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// アクティビティを持つ死亡エンティティを作成
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "テスト敵"})
	world.Components.Dead.Add(enemy, &gc.Dead{})

	comp := activity.NewActivity(gc.BehaviorMelee, 1)
	comp.State = gc.ActivityStateRunning
	world.Components.Activity.Add(enemy, comp)

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// エンティティが削除され、アクティビティも消えている
	assert.False(t, world.ECS.Alive(enemy),
		"死亡エンティティは削除されるべき")
}

func TestDeadCleanupSystem_壊れた収納propは中身を足元へ出す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 4, 4)
	require.NoError(t, err)
	item, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, crate)
	require.NoError(t, err)
	require.True(t, world.Components.LocationInStorage.Has(item), "初期状態は収納内")

	world.Components.Dead.Add(crate, &gc.Dead{})
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	assert.False(t, world.ECS.Alive(crate), "壊れた収納propは消える")
	assert.True(t, world.ECS.Alive(item), "中身は消えず残る")
	assert.True(t, world.Components.LocationOnField.Has(item), "中身は足元のフィールドへ出る")
	assert.False(t, world.Components.LocationInStorage.Has(item), "収納内位置は解除される")
}

// remainsByCoord は置かれた跡を座標から名前への対応で集める
func remainsByCoord(world w.World) map[consts.Coord[consts.Tile]]i18n.Msgid {
	got := map[consts.Coord[consts.Tile]]i18n.Msgid{}
	q := query.ActiveFilter3[gc.GridElement, gc.Name, gc.Remains](world).Query()
	for q.Next() {
		e := q.Entity()
		got[world.Components.GridElement.Get(e).Coord] = world.Components.Name.Get(e).Name
	}
	return got
}

// setMemberRemains は world の raw の敵の行 id の跡を remainsID にする。nil なら跡を外す。ほかの world と
// 共有する行の表は書き換えず、写しを差し替える
func setMemberRemains(world w.World, id string, remainsID *string) {
	members := slices.Clone(raw.PtrSlice(world.Resources.RawMaster.Members))
	for i := range members {
		if members[i].Id == id {
			members[i].RemainsId = remainsID
		}
	}
	world.Resources.RawMaster.Members = &members
}

// setPropRemains は world の raw の置物の行 id の跡を remainsID にする。ほかの world と共有する行の表は書き換えず、
// 写しを差し替える
func setPropRemains(world w.World, id, remainsID string) {
	props := slices.Clone(raw.PtrSlice(world.Resources.RawMaster.Props))
	for i := range props {
		if props[i].Id == id {
			props[i].RemainsId = &remainsID
		}
	}
	world.Resources.RawMaster.Props = &props
}

func TestDeadCleanupSystem_死んだ敵と壊れた物の座標に行の跡を置く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	setMemberRemains(world, "moss_turtle", nil)

	// 草の上で壊れた椅子。ドロップ表を持たない置物の行から跡を引く
	_, err := lifecycle.SpawnDecor(world, "grass", 3, 3)
	require.NoError(t, err)
	chair, err := lifecycle.SpawnProp(world, "chair", 3, 3)
	require.NoError(t, err)
	world.Components.Dead.Add(chair, &gc.Dead{})
	bat, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
	require.NoError(t, err)
	world.Components.Dead.Add(bat, &gc.Dead{})
	turtle, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 7, Y: 7}, "moss_turtle")
	require.NoError(t, err)
	world.Components.Dead.Add(turtle, &gc.Dead{})
	other := world.ECS.NewEntity()
	world.Components.GridElement.Add(other, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}})
	world.Components.Dead.Add(other, &gc.Dead{})

	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	assert.Equal(t, map[consts.Coord[consts.Tile]]i18n.Msgid{
		{X: 3, Y: 3}: "Wood Splinters",
		{X: 5, Y: 5}: "Blood Stain",
	}, remainsByCoord(world))
}

func TestDeadCleanupSystem_跡のあるマスには重ねて置かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnRemains(world, "stone_rubble", consts.Coord[consts.Tile]{X: 2, Y: 2})
	require.NoError(t, err)
	turtle, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 2, Y: 2}, "moss_turtle")
	require.NoError(t, err)
	world.Components.Dead.Add(turtle, &gc.Dead{})
	// 同じ tick に同じマスで2つ壊れる
	for _, id := range []string{"chair", "refrigerator"} {
		e, err := lifecycle.SpawnProp(world, id, 4, 4)
		require.NoError(t, err)
		world.Components.Dead.Add(e, &gc.Dead{})
	}

	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	got := remainsByCoord(world)
	assert.ElementsMatch(t, []consts.Coord[consts.Tile]{{X: 2, Y: 2}, {X: 4, Y: 4}}, slices.Collect(maps.Keys(got)))
	assert.Equal(t, i18n.Msgid("Stone Rubble"), got[consts.Coord[consts.Tile]{X: 2, Y: 2}], "先にあった跡が残る")
}

func TestDeadCleanupSystem_キューブに置いた設備は跡を置かずに消える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, lifecycle.DeployCube(world, cube))
	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	facility, err := lifecycle.PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.NoError(t, err)
	world.Components.Dead.Add(facility, &gc.Dead{})

	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	assert.False(t, world.ECS.Alive(facility))
	assert.Equal(t, map[consts.Coord[consts.Tile]]i18n.Msgid{}, remainsByCoord(world))
}

func TestDeadCleanupSystem_置物の行が無い置物はエラーを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	chair, err := lifecycle.SpawnProp(world, "chair", 3, 3)
	require.NoError(t, err)
	world.Components.RawID.Get(chair).ID = "no_such_prop"
	world.Components.Dead.Add(chair, &gc.Dead{})

	var keyErr raw.KeyNotFoundError
	require.ErrorAs(t, (&DeadCleanupSystem{}).Update(world), &keyErr)
	assert.Equal(t, raw.KeyNotFoundError{Key: "no_such_prop", Collection: "Props"}, keyErr)
}

func TestDeadCleanupSystem_燃え尽きた火は燃え残りと跡の両方を残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	setPropRemains(world, "fire", "stone_rubble")
	coord := consts.Coord[consts.Tile]{X: 4, Y: 7}
	// 着火と同じく、火の置物に残ターン数を付けて燃やす
	fire, err := lifecycle.SpawnProp(world, "fire", coord.X, coord.Y)
	require.NoError(t, err)
	world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})

	require.NoError(t, (&FireSystem{}).Update(world))
	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	assert.Equal(t, map[consts.Coord[consts.Tile]]i18n.Msgid{coord: "Stone Rubble"}, remainsByCoord(world))
	var ashes []consts.Coord[consts.Tile]
	q := query.ActiveFilter2[gc.GridElement, gc.RawID](world).Query()
	for q.Next() {
		if world.Components.RawID.Get(q.Entity()).ID == "ashes" {
			ashes = append(ashes, world.Components.GridElement.Get(q.Entity()).Coord)
		}
	}
	assert.Equal(t, []consts.Coord[consts.Tile]{coord}, ashes)
}

func TestDeadCleanupSystem_地面の飾りは跡を置かずに消える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	grass, err := lifecycle.SpawnDecor(world, "grass", 3, 3)
	require.NoError(t, err)
	world.Components.Dead.Add(grass, &gc.Dead{})

	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	assert.False(t, world.ECS.Alive(grass))
	assert.Equal(t, map[consts.Coord[consts.Tile]]i18n.Msgid{}, remainsByCoord(world))
}
