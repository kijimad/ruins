package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAttackFromCommandTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// 既存の「スライム」コマンドテーブルを使用する
	enemy := world.ECS.NewEntity()
	world.Components.CommandTable.Add(enemy, &gc.CommandTable{
		Name: "スライム",
	})

	// テスト実行
	attack, weaponName, err := GetAttackFromCommandTable(world, enemy)

	// 検証: 有効な攻撃が返されることを確認する
	require.NoError(t, err)
	assert.NotEmpty(t, weaponName)
	assert.NotNil(t, attack)
	assert.Positive(t, attack.GetDamage())
}

func TestGetAttackFromCommandTable_NoCommandTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// CommandTableを持たないエンティティ
	enemy := world.ECS.NewEntity()

	// テスト実行
	_, _, err := GetAttackFromCommandTable(world, enemy)

	// 検証
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no CommandTable component")
}

func TestGetFireFromWeapon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	weapon := world.ECS.NewEntity()
	world.Components.Name.Add(weapon, &gc.Name{Name: "火炎放射器"})
	world.Components.Fire.Add(weapon, &gc.Fire{
		Damage:      5,
		Accuracy:    80,
		AttackCount: 1,
		Element:     gc.ElementTypeFire,
	})

	fire, name, err := GetFireFromWeapon(world, weapon)
	require.NoError(t, err)
	assert.Equal(t, "火炎放射器", name)
	assert.Equal(t, 5, fire.GetDamage())
	assert.Equal(t, gc.ElementTypeFire, fire.GetElement())
}

func TestGetFireFromWeapon_NoFireComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	weapon := world.ECS.NewEntity()
	world.Components.Name.Add(weapon, &gc.Name{Name: "近接武器"})

	_, _, err := GetFireFromWeapon(world, weapon)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no Fire component")
}

func TestGetFireFromWeapon_NoName(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	weapon := world.ECS.NewEntity()

	_, _, err := GetFireFromWeapon(world, weapon)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no Name component")
}

func TestGetMeleeFromWeapon_NoName(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	weapon := world.ECS.NewEntity()

	_, _, err := GetMeleeFromWeapon(world, weapon)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no Name component")
}

func TestGetMeleeFromWeapon_NoMelee(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	weapon := world.ECS.NewEntity()
	world.Components.Name.Add(weapon, &gc.Name{Name: "防具"})

	_, _, err := GetMeleeFromWeapon(world, weapon)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no Melee component")
}

// 統合テスト: 敵とプレイヤーの攻撃取得が共通のAttackerインターフェースを返す
func TestAttackUnification(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// 敵の攻撃取得
	enemy := world.ECS.NewEntity()
	world.Components.CommandTable.Add(enemy, &gc.CommandTable{Name: "スライム"})
	enemyAttack, enemyWeaponName, err := GetAttackFromCommandTable(world, enemy)
	require.NoError(t, err)

	// 取得した敵の攻撃が有効であることを検証する
	assert.NotEmpty(t, enemyWeaponName)
	assert.NotNil(t, enemyAttack)
	assert.Positive(t, enemyAttack.GetDamage())

	// プレイヤーの武器攻撃取得も同じインターフェースで動作することを確認
	playerWeapon := world.ECS.NewEntity()
	world.Components.Name.Add(playerWeapon, &gc.Name{Name: "体当たり"})
	world.Components.Melee.Add(playerWeapon, &gc.Melee{
		Damage:         1,
		Accuracy:       100,
		AttackCount:    1,
		Element:        gc.ElementTypeNone,
		AttackCategory: gc.AttackFist,
	})
	playerAttack, playerWeaponName, err := GetMeleeFromWeapon(world, playerWeapon)
	require.NoError(t, err)
	assert.Equal(t, "体当たり", playerWeaponName)
	assert.Equal(t, 1, playerAttack.GetDamage())
	assert.Equal(t, gc.ElementTypeNone, playerAttack.GetElement())

	// 両方ともAttackerインターフェースを満たすことを検証する
	assert.Implements(t, (*gc.Attacker)(nil), enemyAttack)
	assert.Implements(t, (*gc.Attacker)(nil), playerAttack)
}
