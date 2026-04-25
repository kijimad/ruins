package activity

import (
	"math"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	iw "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// setupShootingWorld は射撃テスト用のWorldを構築する。
// プレイヤーと敵を配置し、プレイヤーにハンドガンを装備させて弾薬を持たせる
func setupShootingWorld(t *testing.T) (world iw.World, player, enemy, weaponEntity ecs.Entity) {
	t.Helper()
	world = testutil.InitTestWorld(t)

	// プレイヤーを生成
	p, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// ハンドガンを生成して装備
	we, err := worldhelper.SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)
	worldhelper.MoveToEquip(world, we, p, gc.SlotWeapon1)
	world.Resources.Dungeon.SelectedWeaponSlot = 1

	// 弾薬を持たせる
	_, err = worldhelper.SpawnItem(world, "9mm FMJ", 30, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// 敵を生成（射程内）
	e, err := worldhelper.SpawnEnemy(world, 13, 10, "火の玉")
	require.NoError(t, err)

	// 敵の位置を探索済みにする
	world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 13, Y: 10}] = true

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
		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		weapon.Magazine = 0

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

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		we, err := worldhelper.SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		// ハンドガンの最大射程(8)より遠くに配置
		enemy, err := worldhelper.SpawnEnemy(world, 20, 10, "火の玉")
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

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 近接武器（木刀）を装備
		we, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		enemy, err := worldhelper.SpawnEnemy(world, 12, 10, "火の玉")
		require.NoError(t, err)

		sa := &ShootActivity{}
		activity, err := NewActivity(sa, 1)
		require.NoError(t, err)
		activity.Target = &enemy

		err = sa.Validate(activity, player, world)
		assert.ErrorIs(t, err, ErrShootNotRangedWeapon)
	})

	t.Run("射線上に壁があるとエラー", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		// 射線上に壁を配置
		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		wall.AddComponent(world.Components.BlockView, &gc.BlockView{})

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
		player.AddComponent(world.Components.Dead, &gc.Dead{})

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
		enemy.AddComponent(world.Components.Dead, &gc.Dead{})

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

		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		before := weapon.Magazine

		sa := &ShootActivity{}
		comp, err := NewActivity(sa, 1)
		require.NoError(t, err)
		comp.Target = &enemy

		err = sa.DoTurn(comp, player, world)
		require.NoError(t, err)

		assert.Equal(t, before-1, weapon.Magazine)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("ターゲット未設定でキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		sa := &ShootActivity{}
		comp, err := NewActivity(sa, 1)
		require.NoError(t, err)

		err = sa.DoTurn(comp, player, world)
		assert.ErrorIs(t, err, ErrAttackTargetNotSet)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestExecuteShootAction(t *testing.T) {
	t.Parallel()

	t.Run("射撃が即時実行され弾薬が消費される", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, weaponEntity := setupShootingWorld(t)

		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		before := weapon.Magazine

		err := ExecuteShootAction(player, enemy, world)
		require.NoError(t, err)

		// 1ターンアクションなので即時完了し、Activityは残らない
		assert.False(t, player.HasComponent(world.Components.Activity))
		// 弾薬が消費されている
		assert.Equal(t, before-1, weapon.Magazine)
	})

	t.Run("近接武器では射撃しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		we, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		enemy, err := worldhelper.SpawnEnemy(world, 12, 10, "火の玉")
		require.NoError(t, err)

		err = ExecuteShootAction(player, enemy, world)
		require.NoError(t, err) // エラーではなくログに記録される

		// Activityは設定されない
		assert.False(t, player.HasComponent(world.Components.Activity))
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

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		we, err := worldhelper.SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		// ハンドガン最大射程(8)より遠く
		enemy, err := worldhelper.SpawnEnemy(world, 20, 10, "火の玉")
		require.NoError(t, err)

		assert.False(t, CanShootTarget(player, enemy, world))
	})

	t.Run("射線上に壁があると射撃不可", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		wall.AddComponent(world.Components.BlockView, &gc.BlockView{})

		assert.False(t, CanShootTarget(player, enemy, world))
	})

	t.Run("死亡した敵は射撃不可", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)
		enemy.AddComponent(world.Components.Dead, &gc.Dead{})

		assert.False(t, CanShootTarget(player, enemy, world))
	})
}

// === EntityDistance テスト ===

func TestEntityDistance(t *testing.T) {
	t.Parallel()

	t.Run("同じ位置なら距離0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.Manager.NewEntity()
		a.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		b := world.Manager.NewEntity()
		b.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		assert.Equal(t, 0.0, EntityDistance(a, b, world))
	})

	t.Run("水平距離", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.Manager.NewEntity()
		a.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		b := world.Manager.NewEntity()
		b.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 0})

		assert.Equal(t, 3.0, EntityDistance(a, b, world))
	})

	t.Run("斜め距離", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.Manager.NewEntity()
		a.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		b := world.Manager.NewEntity()
		b.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 4})

		assert.Equal(t, 5.0, EntityDistance(a, b, world))
	})

	t.Run("GridElementがないと最大値を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		a := world.Manager.NewEntity()
		b := world.Manager.NewEntity()

		assert.Equal(t, math.MaxFloat64, EntityDistance(a, b, world))
	})
}

// === bresenhamLine テスト ===

func TestBresenhamLine(t *testing.T) {
	t.Parallel()

	t.Run("水平線は始点終点を含まない", func(t *testing.T) {
		t.Parallel()
		points := bresenhamLine(0, 0, 4, 0)
		assert.Len(t, points, 3) // (1,0),(2,0),(3,0)
		for _, p := range points {
			assert.Equal(t, 0, p.Y)
		}
	})

	t.Run("同じ座標なら空", func(t *testing.T) {
		t.Parallel()
		points := bresenhamLine(5, 5, 5, 5)
		assert.Empty(t, points)
	})

	t.Run("隣接は空", func(t *testing.T) {
		t.Parallel()
		points := bresenhamLine(0, 0, 1, 0)
		assert.Empty(t, points)
	})

	t.Run("斜め線", func(t *testing.T) {
		t.Parallel()
		points := bresenhamLine(0, 0, 3, 3)
		assert.NotEmpty(t, points)
		for _, p := range points {
			assert.NotEqual(t, consts.Coord[int]{X: 0, Y: 0}, p)
			assert.NotEqual(t, consts.Coord[int]{X: 3, Y: 3}, p)
		}
	})
}

// === checkLineOfSight テスト ===

func TestCheckLineOfSight(t *testing.T) {
	t.Parallel()

	t.Run("遮蔽物なしなら通過", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 0})

		blocked, coverCount := checkLineOfSight(actor, target, world)
		assert.False(t, blocked)
		assert.Equal(t, 0, coverCount)
	})

	t.Run("壁があるとブロックされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 0})

		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 0})
		wall.AddComponent(world.Components.BlockView, &gc.BlockView{})

		blocked, _ := checkLineOfSight(actor, target, world)
		assert.True(t, blocked)
	})

	t.Run("遮蔽物はカバーとしてカウントされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 0})

		// BlockPassだけ（BlockViewなし）→ 遮蔽物
		cover := world.Manager.NewEntity()
		cover.AddComponent(world.Components.GridElement, &gc.GridElement{X: 2, Y: 0})
		cover.AddComponent(world.Components.BlockPass, &gc.BlockPass{})

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

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 0})

		attack := &gc.Attack{AttackCategory: gc.AttackHandgun}
		mod := calculateRangedHitModifier(actor, target, attack, world)
		assert.Equal(t, 0, mod)
	})

	t.Run("最適射程超過でペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		// ハンドガン: 最適射程3, ペナルティ8%/tile → 距離6なら(6-3)*8=24%ペナルティ
		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 0})

		attack := &gc.Attack{AttackCategory: gc.AttackHandgun}
		mod := calculateRangedHitModifier(actor, target, attack, world)
		assert.Less(t, mod, 0)
	})

	t.Run("遮蔽物があるとさらにペナルティ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 0, Y: 0})
		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 0})

		cover := world.Manager.NewEntity()
		cover.AddComponent(world.Components.GridElement, &gc.GridElement{X: 2, Y: 0})
		cover.AddComponent(world.Components.BlockPass, &gc.BlockPass{})

		attack := &gc.Attack{AttackCategory: gc.AttackHandgun}
		mod := calculateRangedHitModifier(actor, target, attack, world)
		assert.Equal(t, -CoverPenaltyPerObject, mod)
	})
}

// === getEquippedRangedWeapon テスト ===

func TestGetEquippedRangedWeapon(t *testing.T) {
	t.Parallel()

	t.Run("遠距離武器が取得できる", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		weapon, err := getEquippedRangedWeapon(player, world)
		require.NoError(t, err)
		assert.Greater(t, weapon.MagazineSize, 0)
	})

	t.Run("プレイヤー以外はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		npc := world.Manager.NewEntity()
		_, err := getEquippedRangedWeapon(npc, world)
		assert.Error(t, err)
	})

	t.Run("武器未装備でエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		_, err = getEquippedRangedWeapon(player, world)
		assert.ErrorIs(t, err, ErrShootNotRangedWeapon)
	})
}

// === CalculateShootHitRate テスト ===

func TestCalculateShootHitRate(t *testing.T) {
	t.Parallel()

	t.Run("命中率が最低値以上最大値以下", func(t *testing.T) {
		t.Parallel()
		world, player, enemy, _ := setupShootingWorld(t)

		hitRate := CalculateShootHitRate(player, enemy, world)
		assert.GreaterOrEqual(t, hitRate, MinHitRate)
		assert.LessOrEqual(t, hitRate, MaxHitRate)
	})

	t.Run("遠距離ほど命中率が下がる", func(t *testing.T) {
		t.Parallel()
		worldNear, playerNear, _, _ := setupShootingWorld(t)

		// 近い敵（距離3）
		nearEnemy, err := worldhelper.SpawnEnemy(worldNear, 13, 10, "火の玉")
		require.NoError(t, err)
		nearRate := CalculateShootHitRate(playerNear, nearEnemy, worldNear)

		// 遠い敵用のWorldを別に構築
		worldFar, playerFar, _, _ := setupShootingWorld(t)
		farEnemy, err := worldhelper.SpawnEnemy(worldFar, 17, 10, "火の玉")
		require.NoError(t, err)
		farRate := CalculateShootHitRate(playerFar, farEnemy, worldFar)

		assert.Greater(t, nearRate, farRate, "近い方が命中率が高い")
	})
}
