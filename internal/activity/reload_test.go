package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReloadBehavior_Info(t *testing.T) {
	t.Parallel()
	ra := &ReloadBehavior{}
	info := ra.Info()
	assert.Equal(t, "装填", info.Name)
	assert.Equal(t, gc.BehaviorReload, ra.Name())
	assert.True(t, info.Interruptible)
}

func TestReloadBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("正常なリロードが検証を通過する", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		// マガジンを空にする
		fire := world.Components.Fire.Get(weaponEntity)
		fire.Magazine = 0

		ra := &ReloadBehavior{}
		comp := NewActivity(gc.BehaviorReload, 0)

		err := ra.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("マガジン満タンなら不要", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		ra := &ReloadBehavior{}
		comp := NewActivity(gc.BehaviorReload, 0)

		err := ra.Validate(comp, player, world)
		assert.ErrorIs(t, err, ErrReloadNotNeeded)
	})

	t.Run("弾薬なしでエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		we, err := lifecycle.SpawnBackpackItem(world, "ハンドガン", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetWeaponSelection(world).Slot = 1

		// マガジンを空にする（弾薬アイテムは持っていない）
		fire := world.Components.Fire.Get(we)
		fire.Magazine = 0

		ra := &ReloadBehavior{}
		comp := NewActivity(gc.BehaviorReload, 0)

		err = ra.Validate(comp, player, world)
		assert.ErrorIs(t, err, ErrReloadNoAmmo)
	})

	t.Run("近接武器ではリロード不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		we, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetWeaponSelection(world).Slot = 1

		ra := &ReloadBehavior{}
		comp := NewActivity(gc.BehaviorReload, 0)

		err = ra.Validate(comp, player, world)
		assert.ErrorIs(t, err, ErrShootNoFireWeapon)
	})
}

func TestReloadBehavior_Start(t *testing.T) {
	t.Parallel()

	t.Run("必要工数が設定される", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		fire := world.Components.Fire.Get(weaponEntity)
		fire.Magazine = 0

		comp, err := NewReloadActivity(player, world)
		require.NoError(t, err)

		assert.Equal(t, fire.ReloadEffort, comp.Progress.Max)
	})
}

func TestReloadBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("工数蓄積で装填が完了する", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		fire := world.Components.Fire.Get(weaponEntity)
		fire.Magazine = 0

		ra := &ReloadBehavior{}
		comp, err := NewReloadActivity(player, world)
		require.NoError(t, err)

		err = ra.Start(comp, player, world)
		require.NoError(t, err)

		// DoTurnを繰り返してリロード完了させる
		for range comp.Progress.Max + 1 {
			if comp.State == gc.ActivityStateCompleted {
				break
			}
			err = ra.DoTurn(comp, player, world)
			require.NoError(t, err)
		}

		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Positive(t, fire.Magazine)
	})

	t.Run("弾薬が不足していたら持っている分だけ装填する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		we, err := lifecycle.SpawnBackpackItem(world, "ハンドガン", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, we, player, gc.SlotWeapon1)
		query.GetWeaponSelection(world).Slot = 1

		fire := world.Components.Fire.Get(we)
		fire.Magazine = 0

		// 弾薬を2発だけ持たせる
		_, err = lifecycle.SpawnBackpackItem(world, "9mm FMJ", 2)
		require.NoError(t, err)

		ra := &ReloadBehavior{}
		comp, err := NewReloadActivity(player, world)
		require.NoError(t, err)

		err = ra.Start(comp, player, world)
		require.NoError(t, err)

		for range comp.Progress.Max + 1 {
			if comp.State == gc.ActivityStateCompleted {
				break
			}
			err = ra.DoTurn(comp, player, world)
			require.NoError(t, err)
		}

		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 2, fire.Magazine)
	})
}

// TestReloadBehavior_進捗はアクティビティごとに独立する は、装填工数の累積を
// Behavior インスタンスのフィールドではなく gc.Activity 側に持たせる規律を固定する。
// 進捗をインスタンスに置くと、1つのインスタンスへ複数アクティビティを通したとき
// 互いの累積を書き換えてしまう。同時装填の破綻に相当する。
func TestReloadBehavior_進捗はアクティビティごとに独立する(t *testing.T) {
	t.Parallel()
	world, player, _, weaponEntity := setupShootingWorld(t)

	fire := world.Components.Fire.Get(weaponEntity)
	fire.Magazine = 0
	fire.ReloadEffort = 1_000_000 // 1ターンでは完了しない十分な工数にする

	// Behavior を1つ取得する
	b, err := GetBehavior(gc.BehaviorReload)
	require.NoError(t, err)
	ra, ok := b.(*ReloadBehavior)
	require.True(t, ok, "GetBehavior(BehaviorReload) は *ReloadBehavior を返すべき")

	// 同一インスタンスに通す2つの独立したアクティビティを用意する
	comp1, err := NewReloadActivity(player, world)
	require.NoError(t, err)
	require.NoError(t, ra.Start(comp1, player, world))

	comp2, err := NewReloadActivity(player, world)
	require.NoError(t, err)
	require.NoError(t, ra.Start(comp2, player, world))

	// 1ターンあたりの工数。同一アクター・同一武器なので両アクティビティで等しい
	expected := ra.calcEffortPerTurn(player, fire, world)
	require.Positive(t, expected)

	// comp1 を1ターン進める。自分の1ターン分だけ累積し、comp2 へ漏れてはいけない
	require.NoError(t, ra.DoTurn(comp1, player, world))
	assert.Equal(t, expected, comp1.Progress.Current)
	assert.Zero(t, comp2.Progress.Current, "comp1 の進行が comp2 に漏れてはいけない")

	// comp2 を1ターン進める。comp1 の累積を引き継がず、自分の1ターン分だけになるべき。
	// 進捗をインスタンスに置くと comp2 は expected の2倍になり、comp1 も書き換わる
	require.NoError(t, ra.DoTurn(comp2, player, world))
	assert.Equal(t, expected, comp2.Progress.Current, "comp2 は自分の1ターン分だけを累積すべき")
	assert.Equal(t, expected, comp1.Progress.Current, "comp2 の進行が comp1 の進捗を書き換えてはいけない")
}

func TestReloadBehavior_CalcEffortPerTurn(t *testing.T) {
	t.Parallel()

	t.Run("基本工数にDEXが加算される", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		fire, _, err := getEquippedFire(player, world)
		require.NoError(t, err)

		ra := &ReloadBehavior{}
		effort := ra.calcEffortPerTurn(player, fire, world)

		// BaseReloadEffort + DEX + weaponSkill
		assert.Greater(t, effort, BaseReloadEffort)
	})

	t.Run("Abilitiesなしなら基本工数のみ", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		fire, _, err := getEquippedFire(player, world)
		require.NoError(t, err)

		// Abilitiesを削除
		world.Components.Abilities.Remove(player)

		ra := &ReloadBehavior{}
		effort := ra.calcEffortPerTurn(player, fire, world)
		assert.Equal(t, BaseReloadEffort, effort)
	})
}

func TestExecuteReloadAction(t *testing.T) {
	t.Parallel()

	t.Run("正常にリロードアクティビティが設定される", func(t *testing.T) {
		t.Parallel()
		world, player, _, weaponEntity := setupShootingWorld(t)

		fire := world.Components.Fire.Get(weaponEntity)
		fire.Magazine = 0

		err := ExecuteReloadAction(player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.Activity.Has(player))
		activity := world.Components.Activity.Get(player)
		assert.Equal(t, gc.BehaviorReload, activity.BehaviorName)
	})

	t.Run("マガジン満タンではアクティビティが設定されない", func(t *testing.T) {
		t.Parallel()
		world, player, _, _ := setupShootingWorld(t)

		err := ExecuteReloadAction(player, world)
		require.Error(t, err)

		assert.False(t, world.Components.Activity.Has(player))
	})
}
