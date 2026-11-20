package query

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

var (
	// errNoCommandTable は敵がCommandTableを持たないときに返す
	errNoCommandTable = errors.New("enemy has no CommandTable component")
	// errEmptyCommandTable は CommandTable に重みを持つ行が無いときに返す
	errEmptyCommandTable = errors.New("command table has no weighted entry")
	// errWeaponNoName は武器がNameを持たないときに返す
	errWeaponNoName = errors.New("weapon has no Name component")
	// errWeaponNoMelee は武器がMeleeを持たないときに返す
	errWeaponNoMelee = errors.New("weapon has no Melee component")
	// errWeaponNoFire は武器がFireを持たないときに返す
	errWeaponNoFire = errors.New("weapon has no Fire component")
)

// GetAttackFromCommandTable は敵のCommandTableからランダムに攻撃を選択し、Meleeパラメータを返す
func GetAttackFromCommandTable(world w.World, enemyEntity ecs.Entity) (gc.Attacker, i18n.Msgid, error) {
	commandTableComp := world.Components.CommandTable.Get(enemyEntity)
	if commandTableComp == nil {
		return nil, "", errNoCommandTable
	}

	commandTableName := commandTableComp.Name
	commandTable, err := raw.GetCommandTable(world.Resources.RawMaster, commandTableName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get command table: %w", err)
	}

	entry, ok := raw.SelectCommandByWeight(commandTable, world.Resources.Config.RNG)
	if !ok {
		return nil, "", fmt.Errorf("%w: %s", errEmptyCommandTable, commandTableName)
	}
	melee, name, err := raw.CommandEntryMelee(world.Resources.RawMaster, entry)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get command attack: %w", err)
	}
	return melee, name, nil
}

// GetMeleeFromWeapon は武器エンティティからMeleeコンポーネントを取得する
func GetMeleeFromWeapon(world w.World, weaponEntity ecs.Entity) (*gc.Melee, i18n.Msgid, error) {
	nameComp := world.Components.Name.Get(weaponEntity)
	if nameComp == nil {
		return nil, "", errWeaponNoName
	}
	name := nameComp.Name

	meleeComp := world.Components.Melee.Get(weaponEntity)
	if meleeComp == nil {
		return nil, "", fmt.Errorf("%w: %s", errWeaponNoMelee, name)
	}
	return meleeComp, name, nil
}

// GetFireFromWeapon は武器エンティティからFireコンポーネントを取得する
func GetFireFromWeapon(world w.World, weaponEntity ecs.Entity) (*gc.Fire, i18n.Msgid, error) {
	nameComp := world.Components.Name.Get(weaponEntity)
	if nameComp == nil {
		return nil, "", errWeaponNoName
	}
	name := nameComp.Name

	fireComp := world.Components.Fire.Get(weaponEntity)
	if fireComp == nil {
		return nil, "", fmt.Errorf("%w: %s", errWeaponNoFire, name)
	}
	return fireComp, name, nil
}
