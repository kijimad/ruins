package raw

import (
	"encoding/json"
	"fmt"

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

// ValidateDisassemblyReferences は分解定義の産出名がアイテム定義に存在することを検証する。
// スキーマ検証は名前の参照整合を見ないため、ここで補う。実行時のスポーン失敗を
// ロード時エラーに前倒しする
func ValidateDisassemblyReferences(raws oapi.Raws) error {
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
		}
		if def.Bonus == nil {
			return nil
		}
		for _, b := range *def.Bonus {
			if _, ok := itemNames[b.Name]; !ok {
				return fmt.Errorf("%s '%s' の分解ボーナス '%s' がアイテム定義に存在しません", ownerKind, ownerName, b.Name)
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
