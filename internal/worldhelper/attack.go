package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetAttackFromCommandTable は敵のCommandTableからランダムに攻撃を選択する
// 武器エンティティを生成せずに攻撃パラメータを取得する
func GetAttackFromCommandTable(world w.World, enemyEntity ecs.Entity) (*gc.Attack, string, error) {
	// CommandTableコンポーネントを取得
	commandTableComp := world.Components.CommandTable.Get(enemyEntity)
	if commandTableComp == nil {
		return nil, "", fmt.Errorf("enemy has no CommandTable component")
	}

	commandTableName := commandTableComp.(*gc.CommandTable).Name
	rawMaster := world.Resources.RawMaster.(*raw.Master)

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

	// Attackコンポーネントがない場合はエラー
	if weaponSpec.Attack == nil {
		return nil, "", fmt.Errorf("weapon %s has no Attack component", weaponName)
	}

	return weaponSpec.Attack, weaponName, nil
}

// GetAttackFromWeapon は武器エンティティから攻撃パラメータを取得する
// プレイヤーが装備している武器エンティティから攻撃情報を取得する
func GetAttackFromWeapon(world w.World, weaponEntity ecs.Entity) (*gc.Attack, string, error) {
	// Attackコンポーネントを取得
	attackComp := world.Components.Attack.Get(weaponEntity)
	if attackComp == nil {
		return nil, "", fmt.Errorf("weapon has no Attack component")
	}

	// 名前を取得
	nameComp := world.Components.Name.Get(weaponEntity)
	if nameComp == nil {
		return nil, "", fmt.Errorf("weapon has no Name component")
	}

	attack := attackComp.(*gc.Attack)
	weaponName := nameComp.(*gc.Name).Name

	return attack, weaponName, nil
}
