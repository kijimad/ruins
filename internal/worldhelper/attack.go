package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetAttackFromCommandTable は敵のCommandTableからランダムに攻撃を選択し、Meleeパラメータを返す
func GetAttackFromCommandTable(world w.World, enemyEntity ecs.Entity) (gc.Attacker, string, error) {
	// CommandTableコンポーネントを取得
	commandTableComp := world.Components.CommandTable.Get(enemyEntity)
	if commandTableComp == nil {
		return nil, "", fmt.Errorf("enemy has no CommandTable component")
	}

	commandTableName := commandTableComp.(*gc.CommandTable).Name
	rawMaster := world.Resources.RawMaster

	// CommandTableを取得
	commandTable, err := rawMaster.GetCommandTable(commandTableName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get command table: %w", err)
	}

	// 重み付きランダムで武器名を選択
	weaponName, err := commandTable.SelectByWeight(world.Config.RNG)
	if err != nil {
		return nil, "", fmt.Errorf("failed to select weapon: %w", err)
	}
	if weaponName == "" {
		return nil, "", fmt.Errorf("no weapon selected from command table")
	}

	// 武器名からEntitySpecを取得（エンティティは生成しない）
	weaponSpec, err := rawMaster.NewWeaponSpec(weaponName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get weapon spec: %w", err)
	}

	if weaponSpec.Melee == nil {
		return nil, "", fmt.Errorf("weapon %s has no Melee component", weaponName)
	}
	return weaponSpec.Melee, weaponName, nil
}

// GetMeleeFromWeapon は武器エンティティからMeleeコンポーネントを取得する
func GetMeleeFromWeapon(world w.World, weaponEntity ecs.Entity) (*gc.Melee, string, error) {
	nameComp := world.Components.Name.Get(weaponEntity)
	if nameComp == nil {
		return nil, "", fmt.Errorf("weapon has no Name component")
	}
	name := nameComp.(*gc.Name).Name

	meleeComp := world.Components.Melee.Get(weaponEntity)
	if meleeComp == nil {
		return nil, "", fmt.Errorf("weapon %s has no Melee component", name)
	}
	return meleeComp.(*gc.Melee), name, nil
}

// GetFireFromWeapon は武器エンティティからFireコンポーネントを取得する
func GetFireFromWeapon(world w.World, weaponEntity ecs.Entity) (*gc.Fire, string, error) {
	nameComp := world.Components.Name.Get(weaponEntity)
	if nameComp == nil {
		return nil, "", fmt.Errorf("weapon has no Name component")
	}
	name := nameComp.(*gc.Name).Name

	fireComp := world.Components.Fire.Get(weaponEntity)
	if fireComp == nil {
		return nil, "", fmt.Errorf("weapon %s has no Fire component", name)
	}
	return fireComp.(*gc.Fire), name, nil
}
