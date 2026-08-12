package raw

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// 参照整合の検証エラー。呼び出し側とテストが errors.Is で種類を同定できるよう sentinel にする。
var (
	errItemTableRefUndefinedGroup     = errors.New("item table references undefined item group")
	errItemGroupRefUndefinedItem      = errors.New("item group references undefined item")
	errEnemyTableRefUndefinedEnemy    = errors.New("enemy table references undefined enemy")
	errCommandTableRefUndefinedWeapon = errors.New("command table references undefined weapon")
	errDropTableMaterialUndefined     = errors.New("drop table references undefined material")
	errMemberDropTableUndefined       = errors.New("member references undefined drop table")
	errMemberCommandTableUndefined    = errors.New("member references undefined command table")
	errDisassemblyYieldUndefined      = errors.New("disassembly yield references undefined item")
	errDisassemblyBonusUndefined      = errors.New("disassembly bonus references undefined item")
	errInvalidPackNotation            = errors.New("invalid pack notation")
	errInvalidLootCountNotation       = errors.New("invalid lootCount notation")
)

// ValidateRaws はoapi.RawsをOpenAPIスキーマの VisitJSON で一括検証する
func ValidateRaws(raws oapi.Raws) error {
	spec, err := oapi.GetSpec()
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}

	schemaRef, ok := spec.Components.Schemas["Raws"]
	if !ok {
		return fmt.Errorf("cannot find Raws component in OpenAPI schema")
	}
	schema := schemaRef.Value
	if schema == nil {
		return fmt.Errorf("got nil Raws schema value")
	}

	jsonBytes, err := json.Marshal(raws)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	var jsonData any
	if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := schema.VisitJSON(jsonData); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return nil
}

// ValidateReferences は定義間の名前参照の整合を検証する。
// スキーマ検証は名前の参照整合を見ないため、ここで補う。実行時の解決失敗を
// ロード時エラーに前倒しする
func ValidateReferences(raws oapi.Raws) error {
	if err := validateDisassemblyReferences(raws); err != nil {
		return err
	}
	if err := validateDropTableReferences(raws); err != nil {
		return err
	}
	if err := validateSpawnDice(raws); err != nil {
		return err
	}
	if err := validateCommandTableReferences(raws); err != nil {
		return err
	}
	if err := validateItemTableReferences(raws); err != nil {
		return err
	}
	if err := validateItemGroupReferences(raws); err != nil {
		return err
	}
	if err := validateEnemyTableReferences(raws); err != nil {
		return err
	}
	return validateCommandTableWeaponReferences(raws)
}

// validateItemTableReferences はアイテムテーブルの参照グループ id がアイテムグループ定義に存在することを検証する。
// 空文字は参照なしとして扱い、非空の参照だけを検証する
func validateItemTableReferences(raws oapi.Raws) error {
	groups := PtrSlice(raws.ItemGroups)
	groupNames := make(map[string]struct{}, len(groups))
	for i := range groups {
		groupNames[groups[i].Id] = struct{}{}
	}

	itemTables := PtrSlice(raws.ItemTables)
	for i := range itemTables {
		for _, entry := range itemTables[i].Entries {
			if entry.Id == "" {
				continue
			}
			if _, ok := groupNames[entry.Id]; !ok {
				return fmt.Errorf("item table %q references group %q: %w", itemTables[i].Name, entry.Id, errItemTableRefUndefinedGroup)
			}
		}
	}
	return nil
}

// validateItemGroupReferences はアイテムグループの参照アイテム id がアイテム定義に存在することを検証する
func validateItemGroupReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	groups := PtrSlice(raws.ItemGroups)
	for i := range groups {
		for _, entry := range groups[i].Entries {
			if entry.Id == "" {
				continue
			}
			if _, ok := itemNames[entry.Id]; !ok {
				return fmt.Errorf("item group %q references item %q: %w", groups[i].Name, entry.Id, errItemGroupRefUndefinedItem)
			}
		}
	}
	return nil
}

// validateEnemyTableReferences は敵テーブルの参照敵 id がメンバー定義に存在することを検証する
func validateEnemyTableReferences(raws oapi.Raws) error {
	members := PtrSlice(raws.Members)
	memberNames := make(map[string]struct{}, len(members))
	for i := range members {
		memberNames[members[i].Id] = struct{}{}
	}

	enemyTables := PtrSlice(raws.EnemyTables)
	for i := range enemyTables {
		for _, entry := range enemyTables[i].Entries {
			if entry.Id == "" {
				continue
			}
			if _, ok := memberNames[entry.Id]; !ok {
				return fmt.Errorf("enemy table %q references enemy %q: %w", enemyTables[i].Name, entry.Id, errEnemyTableRefUndefinedEnemy)
			}
		}
	}
	return nil
}

// validateCommandTableWeaponReferences はコマンドテーブルの参照武器名がアイテム定義に存在することを検証する。
// タイプミスは attack.go の getAttackParams が素手攻撃へ握り潰し無音で劣化するため、ロード時に前倒しで弾く。
func validateCommandTableWeaponReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	commandTables := PtrSlice(raws.CommandTables)
	for i := range commandTables {
		for _, entry := range commandTables[i].Entries {
			if entry.Weapon == "" {
				continue
			}
			if _, ok := itemNames[entry.Weapon]; !ok {
				return fmt.Errorf("command table %q references weapon %q: %w", commandTables[i].Name, entry.Weapon, errCommandTableRefUndefinedWeapon)
			}
		}
	}
	return nil
}

// validateSpawnDice はスポーン系のダイス表記をロード時に検証する。スキーマの pattern は
// "0d6" のような個数0を通すが ParseDice は弾くため、生成時でなくロード時にまとめて弾いて
// 分解産出の count 検証と一貫させる。
func validateSpawnDice(raws oapi.Raws) error {
	enemyTables := PtrSlice(raws.EnemyTables)
	for i := range enemyTables {
		for _, e := range enemyTables[i].Entries {
			if _, err := consts.ParseDice(e.Pack); err != nil {
				return fmt.Errorf("enemy table %q entry %q has %w: %w", enemyTables[i].Name, e.Id, errInvalidPackNotation, err)
			}
		}
	}
	itemGroups := PtrSlice(raws.ItemGroups)
	for i := range itemGroups {
		for _, e := range itemGroups[i].Entries {
			if _, err := consts.ParseDice(e.Pack); err != nil {
				return fmt.Errorf("item group %q entry %q has %w: %w", itemGroups[i].Name, e.Id, errInvalidPackNotation, err)
			}
		}
	}
	props := PtrSlice(raws.Props)
	for i := range props {
		if props[i].Storage == nil || props[i].Storage.LootCount == nil {
			continue
		}
		if _, err := consts.ParseDice(*props[i].Storage.LootCount); err != nil {
			return fmt.Errorf("container %q has %w: %w", props[i].Name, errInvalidLootCountNotation, err)
		}
	}
	return nil
}

// validateDisassemblyReferences は分解定義の産出名がアイテム定義に存在することを検証する
func validateDisassemblyReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	check := func(ownerKind string, ownerName string, def *oapi.Disassembly) error {
		if def == nil {
			return nil
		}
		for _, y := range def.Yields {
			if _, ok := itemNames[y.Id]; !ok {
				return fmt.Errorf("%s %q disassembly yield %q: %w", ownerKind, ownerName, y.Id, errDisassemblyYieldUndefined)
			}
			if _, err := consts.ParseDice(y.Count); err != nil {
				return fmt.Errorf("%s '%s' disassembly yield '%s' has invalid count notation: %w", ownerKind, ownerName, y.Id, err)
			}
		}
		if def.Bonus == nil {
			return nil
		}
		for _, b := range *def.Bonus {
			if _, ok := itemNames[b.Id]; !ok {
				return fmt.Errorf("%s %q disassembly bonus %q: %w", ownerKind, ownerName, b.Id, errDisassemblyBonusUndefined)
			}
			if _, err := consts.ParseDice(b.Count); err != nil {
				return fmt.Errorf("%s '%s' disassembly bonus '%s' has invalid count notation: %w", ownerKind, ownerName, b.Id, err)
			}
		}
		return nil
	}

	props := PtrSlice(raws.Props)
	for i := range props {
		if err := check("prop", props[i].Name, props[i].Disassembly); err != nil {
			return err
		}
	}
	for i := range items {
		if err := check("item", items[i].Name, items[i].Disassembly); err != nil {
			return err
		}
	}
	return nil
}

// validateDropTableReferences はドロップテーブルの素材 id がアイテム定義に存在すること、
// メンバーの dropTableId がテーブル定義に存在することを検証する
func validateDropTableReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	dropTables := PtrSlice(raws.DropTables)
	tableNames := make(map[string]struct{}, len(dropTables))
	for i := range dropTables {
		tableNames[dropTables[i].Id] = struct{}{}
		for _, entry := range dropTables[i].Entries {
			// 空文字はドロップなしを意味する正規の値
			if entry.Material == "" {
				continue
			}
			if _, ok := itemNames[entry.Material]; !ok {
				return fmt.Errorf("drop table %q material %q: %w", dropTables[i].Name, entry.Material, errDropTableMaterialUndefined)
			}
		}
	}

	members := PtrSlice(raws.Members)
	for i := range members {
		// 省略時は参照を持たない。空文字はテーブル名として不正なので存在チェックで弾く
		if members[i].DropTableId == nil {
			continue
		}
		if _, ok := tableNames[*members[i].DropTableId]; !ok {
			return fmt.Errorf("member %q drop table %q: %w", members[i].Name, *members[i].DropTableId, errMemberDropTableUndefined)
		}
	}
	return nil
}

// validateCommandTableReferences はメンバーの commandTableId がテーブル定義に存在することを検証する
func validateCommandTableReferences(raws oapi.Raws) error {
	commandTables := PtrSlice(raws.CommandTables)
	tableNames := make(map[string]struct{}, len(commandTables))
	for i := range commandTables {
		tableNames[commandTables[i].Id] = struct{}{}
	}

	members := PtrSlice(raws.Members)
	for i := range members {
		// 省略時は参照を持たない。空文字はテーブル名として不正なので存在チェックで弾く
		if members[i].CommandTableId == nil {
			continue
		}
		if _, ok := tableNames[*members[i].CommandTableId]; !ok {
			return fmt.Errorf("member %q command table %q: %w", members[i].Name, *members[i].CommandTableId, errMemberCommandTableUndefined)
		}
	}
	return nil
}
