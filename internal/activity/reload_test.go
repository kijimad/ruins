package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloadActivity_Info(t *testing.T) {
	t.Parallel()
	ra := &ReloadActivity{}
	info := ra.Info()
	assert.Equal(t, "装填", info.Name)
	assert.Equal(t, gc.BehaviorReload, ra.Name())
	assert.True(t, info.Interruptible)
}

func TestReloadActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("正常なリロードが検証を通過する", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		// マガジンを空にする
		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		weapon.Magazine = 0

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("マガジン満タンなら不要", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Validate(comp, player, world)
		assert.ErrorIs(t, err, ErrReloadNotNeeded)
	})

	t.Run("弾薬なしでエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		we, err := worldhelper.SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		// マガジンを空にする（弾薬アイテムは持っていない）
		weapon := world.Components.Weapon.Get(we).(*gc.Weapon)
		weapon.Magazine = 0

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Validate(comp, player, world)
		assert.ErrorIs(t, err, ErrReloadNoAmmo)
	})

	t.Run("近接武器ではリロード不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		we, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Validate(comp, player, world)
		assert.ErrorIs(t, err, ErrShootNotRangedWeapon)
	})
}

func TestReloadActivity_Start(t *testing.T) {
	t.Parallel()

	t.Run("ターン数が設定される", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		weapon.Magazine = 0

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Start(comp, player, world)
		require.NoError(t, err)

		assert.Greater(t, comp.TurnsTotal, 0)
		assert.Equal(t, comp.TurnsTotal, comp.TurnsLeft)
	})
}

func TestReloadActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("工数蓄積で装填が完了する", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		weapon.Magazine = 0

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Start(comp, player, world)
		require.NoError(t, err)

		// DoTurnを繰り返してリロード完了させる
		for i := 0; i < comp.TurnsTotal+1; i++ {
			if comp.State == gc.ActivityStateCompleted {
				break
			}
			err = ra.DoTurn(comp, player, world)
			require.NoError(t, err)
		}

		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Greater(t, weapon.Magazine, 0)
	})

	t.Run("弾薬が不足していたら持っている分だけ装填する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		we, err := worldhelper.SpawnItem(world, "ハンドガン", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)
		worldhelper.MoveToEquip(world, we, player, gc.SlotWeapon1)
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		weapon := world.Components.Weapon.Get(we).(*gc.Weapon)
		weapon.Magazine = 0

		// 弾薬を2発だけ持たせる
		_, err = worldhelper.SpawnItem(world, "9mm FMJ", 2, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		ra := &ReloadActivity{}
		comp, err := NewActivity(ra, 1)
		require.NoError(t, err)

		err = ra.Start(comp, player, world)
		require.NoError(t, err)

		for i := 0; i < comp.TurnsTotal+1; i++ {
			if comp.State == gc.ActivityStateCompleted {
				break
			}
			err = ra.DoTurn(comp, player, world)
			require.NoError(t, err)
		}

		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 2, weapon.Magazine)
	})
}

func TestReloadActivity_CalcEffortPerTurn(t *testing.T) {
	t.Parallel()

	t.Run("基本工数にDEXが加算される", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		ra := &ReloadActivity{}
		effort := ra.calcEffortPerTurn(player, world)

		// BaseReloadEffort + DEX + weaponSkill
		assert.Greater(t, effort, BaseReloadEffort)
	})

	t.Run("Abilitiesなしなら基本工数のみ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Player, &gc.Player{})
		world.Resources.Dungeon.SelectedWeaponSlot = 1

		ra := &ReloadActivity{}
		effort := ra.calcEffortPerTurn(actor, world)
		assert.Equal(t, BaseReloadEffort, effort)
	})
}

func TestExecuteReloadAction(t *testing.T) {
	t.Parallel()

	t.Run("正常にリロードアクティビティが設定される", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		weapon := world.Components.Weapon.Get(weaponEntity).(*gc.Weapon)
		weapon.Magazine = 0

		err := ExecuteReloadAction(player, world)
		require.NoError(t, err)

		assert.True(t, player.HasComponent(world.Components.Activity))
		activity := world.Components.Activity.Get(player).(*gc.Activity)
		assert.Equal(t, gc.BehaviorReload, activity.BehaviorName)
	})

	t.Run("マガジン満タンではアクティビティが設定されない", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		err := ExecuteReloadAction(player, world)
		require.NoError(t, err)

		// Activityは設定されない（検証失敗でログに記録される）
		assert.False(t, player.HasComponent(world.Components.Activity))
	})
}
