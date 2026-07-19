package activity

import (
	"math"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/testutil"
	iw "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupShootingWorld は射撃テスト用のWorldを構築する。
// プレイヤーと敵を配置し、プレイヤーにハンドガンを装備させて弾薬を持たせる
func setupShootingWorld(t *testing.T) (world iw.World, player, enemy, weaponEntity ecs.Entity) {
	t.Helper()
	world = testutil.InitTestWorld(t)

	// プレイヤーを生成
	p, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)

	// ハンドガンを生成して装備
	we, err := lifecycle.SpawnBackpackItem(world, "ハンドガン", 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, we, p, gc.SlotWeapon1)
	query.GetDungeon(world).SelectedWeaponSlot = 1

	// 弾薬を持たせる
	_, err = lifecycle.SpawnBackpackItem(world, "9mm FMJ", 30)
	require.NoError(t, err)

	// 敵を生成（射程内）
	e, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "火の玉")
	require.NoError(t, err)

	// 敵の位置を探索済みにする
	query.GetDungeon(world).ExploredTiles[gc.GridElement{X: 13, Y: 10}] = true

	return world, p, e, we
}

// === ShootActivity テスト ===

func TestShootActivity_Info(t *testing.T) {
	t.Parallel()
	sa := &ShootActivity{}
	info := sa.Info()
	assert.Equal(t, "射撃", info.Name)
	assert.Equal(t, gc.BehaviorShoot, sa.Name())
	assert.False(t, info.Interruptible)
}

func TestShootActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("正常な射撃が検証を通過する", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.NoError(t, err)
	})

	t.Run("ターゲット未設定でエラー", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrAttackTargetNotSet)
	})

	t.Run("弾切れでエラー", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, weaponEntity := setupShootingWorld(t)

		// マガジンを空にする
		fire := world.Components.Fire.Get(weaponEntity)
		fire.Magazine = 0

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrShootNoAmmo)
	})

	t.Run("射程外でエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		we, err := lifecycle.SpawnBackpackItem(world, "ハンドガン", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetDungeon(world).SelectedWeaponSlot = 1

		// ハンドガンの最大射程(8)より遠くに配置
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 20, Y: 10}, "火の玉")
		require.NoError(t, err)

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrAttackOutOfRange)
	})

	t.Run("近接武器でエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		// 近接武器（木刀）を装備
		we, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetDungeon(world).SelectedWeaponSlot = 1

		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "火の玉")
		require.NoError(t, err)

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrShootNoFireWeapon)
	})

	t.Run("射線上に壁があるとエラー", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		// 射線上に壁を配置
		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{X: 11, Y: 10})
		world.Components.BlockView.Add(wall, &gc.BlockView{})

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrShootLineOfSightBlocked)
	})

	t.Run("死亡した攻撃者はエラー", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)
		world.Components.Dead.Add(player, &gc.Dead{})

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrAttackerDead)
	})

	t.Run("死亡したターゲットはエラー", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)
		world.Components.Dead.Add(enemy, &gc.Dead{})

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrAttackTargetDead)
	})
}

func TestShootActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("弾薬が1消費される", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, weaponEntity := setupShootingWorld(t)

		fire := world.Components.Fire.Get(weaponEntity)
		before := fire.Magazine

		sa := &ShootActivity{}
		comp, err := NewActivity(sa, 1)
		require.NoError(t, err)
		comp.Target = &enemy

		err = sa.DoTurn(comp, player, world)
		require.NoError(t, err)

		assert.Equal(t, before-1, fire.Magazine)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("ターゲット未設定でキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		sa := &ShootActivity{}
		comp, err := NewActivity(sa, 1)
		require.NoError(t, err)

		err = sa.DoTurn(comp, player, world)
		require.ErrorIs(t, err, ErrAttackTargetNotSet)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestExecuteShootAction(t *testing.T) {
	t.Parallel()

	t.Run("射撃が即時実行され弾薬が消費される", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, weaponEntity := setupShootingWorld(t)

		fire := world.Components.Fire.Get(weaponEntity)
		before := fire.Magazine

		err := ExecuteShootAction(player, enemy, world)
		require.NoError(t, err)

		// 1ターンアクションなので即時完了し、Activityは残らない
		assert.False(t, world.Components.Activity.Has(player))
		// 弾薬が消費されている
		assert.Equal(t, before-1, fire.Magazine)
	})

	t.Run("近接武器では射撃しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		we, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetDungeon(world).SelectedWeaponSlot = 1

		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "火の玉")
		require.NoError(t, err)

		err = ExecuteShootAction(player, enemy, world)
		require.Error(t, err)

		assert.False(t, world.Components.Activity.Has(player))
	})
}

// === CanShootTarget テスト ===

func TestCanShootTarget(t *testing.T) {
	t.Parallel()

	t.Run("射程内の敵は射撃可能", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		assert.True(t, CanShootTarget(player, enemy, world))
	})

	t.Run("射程外の敵は射撃不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		we, err := lifecycle.SpawnBackpackItem(world, "ハンドガン", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetDungeon(world).SelectedWeaponSlot = 1

		// ハンドガン最大射程(8)より遠く
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 20, Y: 10}, "火の玉")
		require.NoError(t, err)

		assert.False(t, CanShootTarget(player, enemy, world))
	})

	t.Run("射線上に壁があると射撃不可", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{X: 11, Y: 10})
		world.Components.BlockView.Add(wall, &gc.BlockView{})

		assert.False(t, CanShootTarget(player, enemy, world))
	})

	t.Run("死亡した敵は射撃不可", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)
		world.Components.Dead.Add(enemy, &gc.Dead{})

		assert.False(t, CanShootTarget(player, enemy, world))
	})
}

// === EntityDistance テスト ===

func TestEntityDistance(t *testing.T) {
	t.Parallel()

	t.Run("同じ位置なら距離0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.ECS.NewEntity()
		world.Components.GridElement.Add(a, &gc.GridElement{X: 5, Y: 5})
		b := world.ECS.NewEntity()
		world.Components.GridElement.Add(b, &gc.GridElement{X: 5, Y: 5})

		assert.Equal(t, 0.0, EntityDistance(a, b, world))
	})

	t.Run("水平距離", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.ECS.NewEntity()
		world.Components.GridElement.Add(a, &gc.GridElement{X: 0, Y: 0})
		b := world.ECS.NewEntity()
		world.Components.GridElement.Add(b, &gc.GridElement{X: 3, Y: 0})

		assert.Equal(t, 3.0, EntityDistance(a, b, world))
	})

	t.Run("斜め距離", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.ECS.NewEntity()
		world.Components.GridElement.Add(a, &gc.GridElement{X: 0, Y: 0})
		b := world.ECS.NewEntity()
		world.Components.GridElement.Add(b, &gc.GridElement{X: 3, Y: 4})

		assert.Equal(t, 5.0, EntityDistance(a, b, world))
	})

	t.Run("GridElementがないと最大値を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.ECS.NewEntity()
		b := world.ECS.NewEntity()

		assert.Equal(t, math.MaxFloat64, EntityDistance(a, b, world))
	})
}

// === checkLineOfSight テスト ===

func TestCheckLineOfSight(t *testing.T) {
	t.Parallel()

	t.Run("遮蔽物なしなら通過", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.GridElement.Add(actor, &gc.GridElement{X: 0, Y: 0})
		target := world.ECS.NewEntity()
		world.Components.GridElement.Add(target, &gc.GridElement{X: 5, Y: 0})

		blocked, coverCount := checkLineOfSight(actor, target, world)
		assert.False(t, blocked)
		assert.Equal(t, 0, coverCount)
	})

	t.Run("壁があるとブロックされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.GridElement.Add(actor, &gc.GridElement{X: 0, Y: 0})
		target := world.ECS.NewEntity()
		world.Components.GridElement.Add(target, &gc.GridElement{X: 5, Y: 0})

		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{X: 3, Y: 0})
		world.Components.BlockView.Add(wall, &gc.BlockView{})

		blocked, _ := checkLineOfSight(actor, target, world)
		assert.True(t, blocked)
	})

	t.Run("遮蔽物はカバーとしてカウントされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.GridElement.Add(actor, &gc.GridElement{X: 0, Y: 0})
		target := world.ECS.NewEntity()
		world.Components.GridElement.Add(target, &gc.GridElement{X: 5, Y: 0})

		// BlockPassだけ（BlockViewなし）→ 遮蔽物
		cover := world.ECS.NewEntity()
		world.Components.GridElement.Add(cover, &gc.GridElement{X: 2, Y: 0})
		world.Components.BlockPass.Add(cover, &gc.BlockPass{})

		blocked, coverCount := checkLineOfSight(actor, target, world)
		assert.False(t, blocked)
		assert.Equal(t, 1, coverCount)
	})
}

// === calculateRangedHitModifier テスト ===

func TestCalculateRangedHitModifier(t *testing.T) {
	t.Parallel()

	t.Run("最適射程内はペナルティなし", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.GridElement.Add(actor, &gc.GridElement{X: 0, Y: 0})
		target := world.ECS.NewEntity()
		world.Components.GridElement.Add(target, &gc.GridElement{X: 3, Y: 0})

		fire := &gc.Fire{AttackCategory: gc.AttackHandgun}
		mod := calculateRangedHitModifier(actor, target, fire, world)
		assert.Equal(t, 0, mod)
	})

	t.Run("最適射程超過でペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.GridElement.Add(actor, &gc.GridElement{X: 0, Y: 0})
		// ハンドガン: 最適射程3, ペナルティ8%/tile → 距離6なら(6-3)*8=24%ペナルティ
		target := world.ECS.NewEntity()
		world.Components.GridElement.Add(target, &gc.GridElement{X: 6, Y: 0})

		fire := &gc.Fire{AttackCategory: gc.AttackHandgun}
		mod := calculateRangedHitModifier(actor, target, fire, world)
		assert.Negative(t, mod)
	})

	t.Run("遮蔽物があるとさらにペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.GridElement.Add(actor, &gc.GridElement{X: 0, Y: 0})
		target := world.ECS.NewEntity()
		world.Components.GridElement.Add(target, &gc.GridElement{X: 3, Y: 0})

		cover := world.ECS.NewEntity()
		world.Components.GridElement.Add(cover, &gc.GridElement{X: 2, Y: 0})
		world.Components.BlockPass.Add(cover, &gc.BlockPass{})

		fire := &gc.Fire{AttackCategory: gc.AttackHandgun}
		mod := calculateRangedHitModifier(actor, target, fire, world)
		assert.Equal(t, -CoverPenaltyPerObject, mod)
	})
}

// === getEquippedFire テスト ===

func TestGetEquippedFire(t *testing.T) {
	t.Parallel()

	t.Run("遠距離武器が取得できる", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		fire, _, err := getEquippedFire(player, world)
		require.NoError(t, err)
		assert.Positive(t, fire.MagazineSize)
	})

	t.Run("武器未装備でエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		query.GetDungeon(world).SelectedWeaponSlot = 1

		_, _, err = getEquippedFire(player, world)
		assert.ErrorIs(t, err, ErrShootNoFireWeapon)
	})
}

// === CalculateShootHitRate テスト ===

func TestCalculateShootHitRate(t *testing.T) {
	t.Parallel()

	t.Run("命中率が最低値以上最大値以下", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		hitRate := CalculateShootHitRate(player, enemy, world)
		assert.GreaterOrEqual(t, hitRate, formula.MinHitRate)
		assert.LessOrEqual(t, hitRate, formula.MaxHitRate)
	})

	t.Run("遠距離ほど命中率が下がる", func(t *testing.T) {
		t.Parallel()
		worldNear, playerNear, _, _ := setupShootingWorld(t)

		// 近い敵（距離3）
		nearEnemy, err := lifecycle.SpawnEnemy(worldNear, consts.Coord[consts.Tile]{X: 13, Y: 10}, "火の玉")
		require.NoError(t, err)
		nearRate := CalculateShootHitRate(playerNear, nearEnemy, worldNear)

		// 遠い敵用のWorldを別に構築
		worldFar, playerFar, _, _ := setupShootingWorld(t)
		farEnemy, err := lifecycle.SpawnEnemy(worldFar, consts.Coord[consts.Tile]{X: 17, Y: 10}, "火の玉")
		require.NoError(t, err)
		farRate := CalculateShootHitRate(playerFar, farEnemy, worldFar)

		assert.Greater(t, nearRate, farRate, "近い方が命中率が高い")
	})
}
