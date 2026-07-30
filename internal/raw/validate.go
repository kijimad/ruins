package raw

import (
	"encoding/json"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// ValidateRaws はoapi.RawsをOpenAPIスキーマの VisitJSON で一括検証する
func ValidateRaws(raws oapi.Raws) error {
	spec, err := oapi.GetSpec()
	if err != nil {
		return fmt.Errorf("OpenAPIスキーマの読み込みに失敗: %w", err)
	}

	schemaRef, ok := spec.Components.Schemas["Raws"]
	if !ok {
		return fmt.Errorf("OpenAPIスキーマにRawsコンポーネントが見つからない")
	}
	schema := schemaRef.Value
	if schema == nil {
		return fmt.Errorf("Rawsスキーマの値がnil")
	}

	jsonBytes, err := json.Marshal(raws)
	if err != nil {
		return fmt.Errorf("JSONマーシャルに失敗: %w", err)
	}

	var jsonData any
	if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return fmt.Errorf("JSONアンマーシャルに失敗: %w", err)
	}

	if err := schema.VisitJSON(jsonData); err != nil {
		return fmt.Errorf("バリデーションエラー: %w", err)
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
	return validateCommandTableReferences(raws)
}

// validateSpawnDice はスポーン系のダイス表記をロード時に検証する。スキーマの pattern は
// "0d6" のような個数0を通すが ParseDice は弾くため、生成時でなくロード時にまとめて弾いて
// 分解産出の count 検証と一貫させる。
func validateSpawnDice(raws oapi.Raws) error {
	enemyTables := PtrSlice(raws.EnemyTables)
	for i := range enemyTables {
		for _, e := range enemyTables[i].Entries {
			if _, err := consts.ParseDice(e.Pack); err != nil {
				return fmt.Errorf("敵テーブル '%s' の '%s' のパック表記が不正です: %w", enemyTables[i].Name, e.EnemyName, err)
			}
		}
	}
	itemGroups := PtrSlice(raws.ItemGroups)
	for i := range itemGroups {
		for _, e := range itemGroups[i].Entries {
			if _, err := consts.ParseDice(e.Pack); err != nil {
				return fmt.Errorf("アイテムグループ '%s' の '%s' のパック表記が不正です: %w", itemGroups[i].Name, e.ItemName, err)
			}
		}
	}
	props := PtrSlice(raws.Props)
	for i := range props {
		if props[i].Storage == nil || props[i].Storage.LootCount == nil {
			continue
		}
		if _, err := consts.ParseDice(*props[i].Storage.LootCount); err != nil {
			return fmt.Errorf("収納 '%s' の lootCount 表記が不正です: %w", props[i].Name, err)
		}
	}
	return nil
}

// validateDisassemblyReferences は分解定義の産出名がアイテム定義に存在することを検証する
func validateDisassemblyReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Name] = struct{}{}
	}

	check := func(ownerKind string, ownerName string, def *oapi.Disassembly) error {
		if def == nil {
			return nil
		}
		for _, y := range def.Yields {
			if _, ok := itemNames[y.Name]; !ok {
				return fmt.Errorf("%s '%s' の分解産出 '%s' がアイテム定義に存在しません", ownerKind, ownerName, y.Name)
			}
			if _, err := consts.ParseDice(y.Count); err != nil {
				return fmt.Errorf("%s '%s' の分解産出 '%s' の個数表記が不正です: %w", ownerKind, ownerName, y.Name, err)
			}
		}
		if def.Bonus == nil {
			return nil
		}
		for _, b := range *def.Bonus {
			if _, ok := itemNames[b.Name]; !ok {
				return fmt.Errorf("%s '%s' の分解ボーナス '%s' がアイテム定義に存在しません", ownerKind, ownerName, b.Name)
			}
			if _, err := consts.ParseDice(b.Count); err != nil {
				return fmt.Errorf("%s '%s' の分解ボーナス '%s' の個数表記が不正です: %w", ownerKind, ownerName, b.Name, err)
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

// validateDropTableReferences はドロップテーブルの素材名がアイテム定義に存在すること、
// メンバーの dropTableName がテーブル定義に存在することを検証する
func validateDropTableReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Name] = struct{}{}
	}

	dropTables := PtrSlice(raws.DropTables)
	tableNames := make(map[string]struct{}, len(dropTables))
	for i := range dropTables {
		tableNames[dropTables[i].Name] = struct{}{}
		for _, entry := range dropTables[i].Entries {
			// 空文字はドロップなしを意味する正規の値
			if entry.Material == "" {
				continue
			}
			if _, ok := itemNames[entry.Material]; !ok {
				return fmt.Errorf("ドロップテーブル '%s' の素材 '%s' がアイテム定義に存在しません", dropTables[i].Name, entry.Material)
			}
		}
	}

	members := PtrSlice(raws.Members)
	for i := range members {
		// 空文字は未指定と同じ扱いにする。EntitySpec 構築側の判定と揃える
		if members[i].DropTableName == nil || *members[i].DropTableName == "" {
			continue
		}
		if _, ok := tableNames[*members[i].DropTableName]; !ok {
			return fmt.Errorf("メンバー '%s' のドロップテーブル '%s' が定義に存在しません", members[i].Name, *members[i].DropTableName)
		}
	}
	return nil
}

// validateCommandTableReferences はメンバーの commandTableName がテーブル定義に存在することを検証する
func validateCommandTableReferences(raws oapi.Raws) error {
	commandTables := PtrSlice(raws.CommandTables)
	tableNames := make(map[string]struct{}, len(commandTables))
	for i := range commandTables {
		tableNames[commandTables[i].Name] = struct{}{}
	}

	members := PtrSlice(raws.Members)
	for i := range members {
		// 空文字は未指定と同じ扱いにする。EntitySpec 構築側の判定と揃える
		if members[i].CommandTableName == nil || *members[i].CommandTableName == "" {
			continue
		}
		if _, ok := tableNames[*members[i].CommandTableName]; !ok {
			return fmt.Errorf("メンバー '%s' のコマンドテーブル '%s' が定義に存在しません", members[i].Name, *members[i].CommandTableName)
		}
	}
	return nil
}
