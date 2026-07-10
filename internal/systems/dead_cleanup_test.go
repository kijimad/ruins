package systems

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/mlange-42/ark/ecs"
)

func TestDeadCleanupSystem(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成

	// 1. 通常の敵（AI）エンティティ - 削除されるべき
	enemy := world.World.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "テスト敵"})
	world.Components.SoloAI.Add(enemy, &gc.SoloAI{})
	world.Components.Dead.Add(enemy, &gc.Dead{})

	// 2. プレイヤーエンティティ - 削除されないべき
	player := world.World.NewEntity()
	world.Components.Name.Add(player, &gc.Name{Name: "プレイヤー"})
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Dead.Add(player, &gc.Dead{})

	// 3. その他のDeadエンティティ - 削除されるべき
	otherDead := world.World.NewEntity()
	world.Components.Name.Add(otherDead, &gc.Name{Name: "その他"})
	world.Components.Dead.Add(otherDead, &gc.Dead{})

	// 4. 生きているエンティティ - 削除されないべき
	alive := world.World.NewEntity()
	world.Components.Name.Add(alive, &gc.Name{Name: "生きている敵"})
	world.Components.SoloAI.Add(alive, &gc.SoloAI{})

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 結果を検証

	// 敵エンティティは削除されているべき（Nameコンポーネントも削除される）
	assert.False(t, world.Components.Name.Has(enemy), "敵エンティティは削除されるべき")

	// プレイヤーエンティティは削除されていないべき
	assert.True(t, world.Components.Name.Has(player), "プレイヤーエンティティは削除されないべき")
	assert.True(t, world.Components.Dead.Has(player), "プレイヤーのDeadコンポーネントは残るべき")

	// その他のDeadエンティティは削除されているべき
	assert.False(t, world.Components.Name.Has(otherDead), "その他のDeadエンティティは削除されるべき")

	// 生きているエンティティは削除されていないべき
	assert.True(t, world.Components.Name.Has(alive), "生きているエンティティは削除されないべき")
	assert.False(t, world.Components.Dead.Has(alive), "生きているエンティティにDeadコンポーネントはないべき")

	// クリーンアップ
	world.World.RemoveEntity(player)
	world.World.RemoveEntity(alive)
}

func TestDeadCleanupSystem_NoDeadEntities(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// Deadエンティティが存在しない状態でテスト
	alive1 := world.World.NewEntity()
	world.Components.Name.Add(alive1, &gc.Name{Name: "生きている1"})

	alive2 := world.World.NewEntity()
	world.Components.Name.Add(alive2, &gc.Name{Name: "生きている2"})
	world.Components.SoloAI.Add(alive2, &gc.SoloAI{})

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// すべてのエンティティが残っているべき
	assert.True(t, world.Components.Name.Has(alive1), "生きているエンティティ1は残るべき")
	assert.True(t, world.Components.Name.Has(alive2), "生きているエンティティ2は残るべき")

	// クリーンアップ
	world.World.RemoveEntity(alive1)
	world.World.RemoveEntity(alive2)
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
	deadQuery := ecs.NewFilter1[gc.Dead](world.World).Query()
	for deadQuery.Next() {
		deadCount++
	}
	assert.Equal(t, 0, deadCount, "Deadコンポーネントを持つエンティティは存在しない")
}

func TestDeadCleanupSystem_WithDropTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// ドロップテーブルを持つ敵エンティティを作成（灰の偶像は100%ドロップ）
	enemy := world.World.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "灰の偶像"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.DropTable.Add(enemy, &gc.DropTable{Name: "灰の偶像"})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: 5, Y: 5})

	// DeadCleanupSystem実行前のアイテムエンティティ数をカウント
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationOnField](world.World).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 敵エンティティは削除されているべき
	assert.False(t, world.Components.Name.Has(enemy), "敵エンティティは削除されるべき")

	// ドロップアイテムが生成されているべき（"鉄くず"がドロップテーブルに定義されている）
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationOnField](world.World).Query()
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
	world.Config.Seed = 2
	world.Config.RNG = rand.New(rand.NewPCG(world.Config.Seed, 0))

	// 敵エンティティを作成
	enemy := world.World.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "火の玉"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.DropTable.Add(enemy, &gc.DropTable{Name: "火の玉"})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: 5, Y: 5})

	// 実行前のアイテム数
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationOnField](world.World).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 実行後のアイテム数
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationOnField](world.World).Query()
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
	enemy := world.World.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "火の玉"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: 5, Y: 5})

	// 実行前のアイテム数
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationOnField](world.World).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 実行後のアイテム数
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationOnField](world.World).Query()
	for itemAfterQuery.Next() {
		itemCountAfter++
	}

	assert.Equal(t, itemCountBefore, itemCountAfter, "ドロップテーブルなしではドロップしない")
}

func TestDeadCleanupSystem_CancelsActivity(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// アクティビティを持つ死亡エンティティを作成
	enemy := world.World.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "テスト敵"})
	world.Components.Dead.Add(enemy, &gc.Dead{})

	aa := &activity.AttackActivity{}
	comp, err := activity.NewActivity(aa, 1)
	require.NoError(t, err)
	comp.State = gc.ActivityStateRunning
	world.Components.Activity.Add(enemy, comp)

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// エンティティが削除され、アクティビティも消えている
	assert.False(t, world.Components.Activity.Has(enemy),
		"死亡エンティティのアクティビティはキャンセルされるべき")
	assert.False(t, world.Components.Name.Has(enemy),
		"死亡エンティティは削除されるべき")
}

func TestDeadCleanupSystem_SpawnsSpriteFadeoutEffect(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// SpriteRenderを持つ敵エンティティを作成
	enemy := world.World.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "スライム"})
	world.Components.Dead.Add(enemy, &gc.Dead{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: 5, Y: 5})
	world.Components.SpriteRender.Add(enemy, &gc.SpriteRender{
		SpriteSheetName: "character",
		SpriteKey:       "slime_0",
	})

	// 実行前のVisualEffectエンティティ数
	effectCountBefore := 0
	effectBeforeQuery := ecs.NewFilter1[gc.VisualEffects](world.World).Query()
	for effectBeforeQuery.Next() {
		effectCountBefore++
	}

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 敵エンティティは削除されているべき
	assert.False(t, world.Components.Name.Has(enemy), "敵エンティティは削除されるべき")

	// スプライトフェードアウトエフェクトが生成されているべき
	effectCountAfter := 0
	effectAfterQuery := ecs.NewFilter1[gc.VisualEffects](world.World).Query()
	for effectAfterQuery.Next() {
		effectCountAfter++
	}
	assert.Equal(t, effectCountBefore+1, effectCountAfter, "スプライトフェードアウトエフェクトが生成されているべき")

	// エフェクトの内容を確認
	effectQuery := ecs.NewFilter2[gc.VisualEffects, gc.GridElement](world.World).Query()
	for effectQuery.Next() {
		entity := effectQuery.Entity()
		ve := world.Components.VisualEffect.Get(entity)
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
