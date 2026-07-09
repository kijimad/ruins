package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetAttackFromCommandTable は敵のCommandTableからランダムに攻撃を選択し、Meleeパラメータを返す
func GetAttackFromCommandTable(world w.World, enemyEntity ecs.Entity) (gc.Attacker, string, error) {
	commandTableComp, ok := world.Components.CommandTable.TryGet(enemyEntity)
	if !ok {
		return nil, "", fmt.Errorf("enemy has no CommandTable component")
	}

	commandTableName := commandTableComp.Name
	commandTable, err := raw.GetCommandTable(world.Resources.RawMaster, commandTableName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get command table: %w", err)
	}

	weaponName, err := raw.SelectCommandByWeight(commandTable, world.Config.RNG)
	if err != nil {
		return nil, "", fmt.Errorf("failed to select weapon: %w", err)
	}
	if weaponName == "" {
		return nil, "", fmt.Errorf("no weapon selected from command table")
	}

	weaponSpec, err := raw.NewWeaponSpec(world.Resources.RawMaster, weaponName)
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
	nameComp, ok := world.Components.Name.TryGet(weaponEntity)
	if !ok {
		return nil, "", fmt.Errorf("weapon has no Name component")
	}
	name := nameComp.Name

	meleeComp, ok := world.Components.Melee.TryGet(weaponEntity)
	if !ok {
		return nil, "", fmt.Errorf("weapon %s has no Melee component", name)
	}
	return meleeComp, name, nil
}

// GetFireFromWeapon は武器エンティティからFireコンポーネントを取得する
func GetFireFromWeapon(world w.World, weaponEntity ecs.Entity) (*gc.Fire, string, error) {
	nameComp, ok := world.Components.Name.TryGet(weaponEntity)
	if !ok {
		return nil, "", fmt.Errorf("weapon has no Name component")
	}
	name := nameComp.Name

	fireComp, ok := world.Components.Fire.TryGet(weaponEntity)
	if !ok {
		return nil, "", fmt.Errorf("weapon %s has no Fire component", name)
	}
	return fireComp, name, nil
}
