package systems

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
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

func TestDeadCleanupSystem_SpawnsSpriteFadeoutEffect(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// SpriteRenderを持つ敵エンティティを作成
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "スライム"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.SpriteRender.Add(enemy, &gc.SpriteRender{
		SpriteSheetName: "character",
		SpriteKey:       "slime_0",
	})

	// 実行前のVisualEffectエンティティ数
	effectCountBefore := 0
	effectBeforeQuery := ecs.NewFilter1[gc.VisualEffects](world.ECS).Query()
	for effectBeforeQuery.Next() {
		effectCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 敵エンティティは削除されているべき
	assert.False(t, world.ECS.Alive(enemy), "敵エンティティは削除されるべき")

	// スプライトフェードアウトエフェクトが生成されているべき
	effectCountAfter := 0
	effectAfterQuery := ecs.NewFilter1[gc.VisualEffects](world.ECS).Query()
	for effectAfterQuery.Next() {
		effectCountAfter++
	}
	assert.Equal(t, effectCountBefore+1, effectCountAfter, "スプライトフェードアウトエフェクトが生成されているべき")

	// エフェクトの内容を確認
	effectQuery := ecs.NewFilter2[gc.VisualEffects, gc.GridElement](world.ECS).Query()
	for effectQuery.Next() {
		entity := effectQuery.Entity()
		ve := world.Components.VisualEffects.Get(entity)
		ge := world.Components.GridElement.Get(entity)

		require.Len(t, ve.Effects, 1)
		effect, ok := ve.Effects[0].(*gc.SpriteFadeoutEffect)
		require.True(t, ok, "SpriteFadeoutEffectであるべき")

		assert.Equal(t, "character", effect.SpriteSheetName)
		assert.Equal(t, "slime_0", effect.SpriteKey)
		assert.Equal(t, consts.Tile(5), ge.X, "エフェクトは敵の位置に生成されるべき")
		assert.Equal(t, consts.Tile(5), ge.Y, "エフェクトは敵の位置に生成されるべき")
	}
}

// TestDeadCleanupSystem_DropsBackpackItems は死亡エンティティのバックパック内アイテムが
// フィールドへドロップされることを確認する。所有者はフィールドへ移送され、アイテムは
// 死亡位置の GridElement を得る。
func TestDeadCleanupSystem_DropsBackpackItems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// バックパックを持つ死亡エンティティ
	deadCoord := consts.Coord[consts.Tile]{X: 4, Y: 6}
	owner := world.ECS.NewEntity()
	world.Components.Name.Add(owner, &gc.Name{Name: "所有者"})
	world.Components.Dead.Add(owner, &gc.Dead{})
	world.Components.GridElement.Add(owner, &gc.GridElement{Coord: deadCoord})

	// 所有者のバックパックに入ったアイテム
	item := world.ECS.NewEntity()
	world.Components.Name.Add(item, &gc.Name{Name: "遺品"})
	world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

	// 別の所有者のアイテムは巻き込まれないことも確認する
	otherOwner := world.ECS.NewEntity()
	other := world.ECS.NewEntity()
	world.Components.Name.Add(other, &gc.Name{Name: "他人の物"})
	world.Components.LocationInBackpack.Add(other, &gc.LocationInBackpack{Owner: otherOwner})

	require.NoError(t, (&DeadCleanupSystem{}).Update(world))

	// 死亡した所有者は削除される
	assert.False(t, world.ECS.Alive(owner), "死亡した所有者は削除されるべき")

	// バックパック内アイテムはフィールドへ移り、死亡位置の座標を得る
	require.True(t, world.ECS.Alive(item), "遺品は残るべき")
	assert.True(t, world.Components.LocationOnField.Has(item), "遺品はフィールドへ移るべき")
	assert.False(t, world.Components.LocationInBackpack.Has(item), "遺品はバックパックから外れるべき")
	require.True(t, world.Components.GridElement.Has(item), "遺品は座標を得るべき")
	itemCoord := world.Components.GridElement.Get(item).Coord
	assert.Equal(t, deadCoord, itemCoord, "遺品は死亡位置に落ちるべき")

	// 生存する別所有者のアイテムはバックパックに残る
	assert.True(t, world.Components.LocationInBackpack.Has(other), "他人の物は巻き込まれないべき")
}

// TestDeadCleanupSystem_DisassemblesProp は分解定義を持つ prop が破壊されたとき、
// 分解産出の回収パスを通ってエンティティが削除されることを確認する。
// barrel は raw で分解定義を持つ prop。
func TestDeadCleanupSystem_DisassemblesProp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// 分解定義を持つ prop を生成し、破壊状態にする
	prop, err := lifecycle.SpawnProp(world, "barrel", 2, 3)
	require.NoError(t, err)
	require.True(t, world.Components.Fixed.Has(prop), "prop は Fixed を持つ前提")
	world.Components.Dead.Add(prop, &gc.Dead{})

	// 分解産出の回収パスを通り、エラーなく prop が削除される
	require.NoError(t, (&DeadCleanupSystem{}).Update(world))
	assert.False(t, world.ECS.Alive(prop), "破壊された prop は削除されるべき")
}
