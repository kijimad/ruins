package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAttackFromCommandTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// テスト用のCommandTableを作成
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	rawMaster.Raws.CommandTables = []raw.CommandTable{
		{
			Name: "test_goblin_attacks",
			Entries: []raw.CommandTableEntry{
				{Weapon: "木刀", Weight: 1.0},
			},
		},
	}
	rawMaster.CommandTableIndex = map[string]int{
		"test_goblin_attacks": 0,
	}

	// 敵エンティティを作成
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.CommandTable, &gc.CommandTable{
		Name: "test_goblin_attacks",
	})

	// テスト実行
	attack, weaponName, err := GetAttackFromCommandTable(world, enemy)

	// 検証
	require.NoError(t, err)
	assert.Equal(t, "木刀", weaponName)
	assert.NotNil(t, attack)
	assert.Equal(t, 8, attack.Damage) // 木刀の実際のダメージ値
}

func TestGetAttackFromCommandTable_NoCommandTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// CommandTableを持たないエンティティ
	enemy := world.Manager.NewEntity()

	// テスト実行
	_, _, err := GetAttackFromCommandTable(world, enemy)

	// 検証
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no CommandTable component")
}

func TestGetAttackFromWeapon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// 武器エンティティを作成
	weapon := world.Manager.NewEntity()
	weapon.AddComponent(world.Components.Name, &gc.Name{Name: "火炎斬り"})
	weapon.AddComponent(world.Components.Attack, &gc.Attack{
		Damage:      20,
		Accuracy:    90,
		AttackCount: 1,
		Element:     gc.ElementTypeFire,
	})

	// テスト実行
	attack, weaponName, err := GetAttackFromWeapon(world, weapon)

	// 検証
	require.NoError(t, err)
	assert.Equal(t, "火炎斬り", weaponName)
	assert.NotNil(t, attack)
	assert.Equal(t, 20, attack.Damage)
	assert.Equal(t, gc.ElementTypeFire, attack.Element)
}

func TestGetAttackFromWeapon_NoAttack(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// Attackコンポーネントを持たないエンティティ
	weapon := world.Manager.NewEntity()
	weapon.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

	// テスト実行
	_, _, err := GetAttackFromWeapon(world, weapon)

	// 検証
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no Attack component")
}

// 統合テスト: 敵とプレイヤーの攻撃取得が共通のAttack構造体を返す
func TestAttackUnification(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	rawMaster := world.Resources.RawMaster.(*raw.Master)
	rawMaster.Raws.CommandTables = []raw.CommandTable{
		{
			Name: "enemy_attacks",
			Entries: []raw.CommandTableEntry{
				{Weapon: "木刀", Weight: 1.0},
			},
		},
	}
	rawMaster.CommandTableIndex = map[string]int{
		"enemy_attacks": 0,
	}

	// 敵の攻撃取得
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.CommandTable, &gc.CommandTable{Name: "enemy_attacks"})
	enemyAttack, enemyWeaponName, err := GetAttackFromCommandTable(world, enemy)
	require.NoError(t, err)

	// プレイヤーの武器攻撃取得
	playerWeapon := world.Manager.NewEntity()
	playerWeapon.AddComponent(world.Components.Name, &gc.Name{Name: "木刀"})
	playerWeapon.AddComponent(world.Components.Attack, &gc.Attack{
		Damage:         8, // 木刀の実際のダメージ値
		Accuracy:       100,
		AttackCount:    1,
		Element:        gc.ElementTypeNone,
		AttackCategory: gc.AttackSword,
	})
	playerAttack, playerWeaponName, err := GetAttackFromWeapon(world, playerWeapon)
	require.NoError(t, err)

	// 同じ武器名で同じ攻撃パラメータを取得できることを確認
	assert.Equal(t, enemyWeaponName, playerWeaponName)
	assert.Equal(t, enemyAttack.Damage, playerAttack.Damage)
	assert.Equal(t, enemyAttack.Element, playerAttack.Element)
}
