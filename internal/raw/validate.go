package raw

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/kijimaD/ruins/internal/oapi"
)

// ValidateRaws はRawsの全エンティティをOpenAPIスキーマでバリデーションする。
// Go構造体をJSONに変換し、対応するOpenAPIスキーマの VisitJSON で検証する。
// Goのゼロ値とTypeSpecのrequired定義の不一致を回避するため、
// スキーマからRequired制約を除去してから検証する
func ValidateRaws(raws Raws) error {
	swagger, err := oapi.GetSwagger()
	if err != nil {
		return fmt.Errorf("OpenAPIスキーマの読み込みに失敗: %w", err)
	}

	type entry struct {
		schemaName string
		items      []any
	}
	entries := []entry{
		{"Item", toAnySlice(raws.Items)},
		{"Recipe", toAnySlice(raws.Recipes)},
		{"Member", toAnySlice(raws.Members)},
		{"CommandTable", toAnySlice(raws.CommandTables)},
		{"DropTable", toAnySlice(raws.DropTables)},
		{"ItemGroup", toAnySlice(raws.ItemGroups)},
		{"ItemTable", toAnySlice(raws.ItemTables)},
		{"EnemyTable", toAnySlice(raws.EnemyTables)},
		{"SpriteSheet", toAnySlice(raws.SpriteSheets)},
		{"Tile", toAnySlice(raws.Tiles)},
		{"Prop", toAnySlice(raws.Props)},
		{"Profession", toAnySlice(raws.Professions)},
	}

	var errs []string
	for _, e := range entries {
		schemaRef, ok := swagger.Components.Schemas[e.schemaName]
		if !ok {
			return fmt.Errorf("OpenAPIスキーマにコンポーネントが見つからない: %s", e.schemaName)
		}
		schema := schemaRef.Value
		if schema == nil {
			return fmt.Errorf("スキーマの値がnil: %s", e.schemaName)
		}

		// Goのゼロ値とTypeSpecのrequired定義が一致しないため、
		// Required制約を除去したスキーマで検証する
		relaxed := relaxSchema(schema, swagger.Components.Schemas)

		for i, item := range e.items {
			jsonBytes, err := json.Marshal(item)
			if err != nil {
				return fmt.Errorf("%s[%d]: JSONマーシャルに失敗: %w", e.schemaName, i, err)
			}

			var jsonData any
			if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
				return fmt.Errorf("%s[%d]: JSONアンマーシャルに失敗: %w", e.schemaName, i, err)
			}

			// Goのnilスライスがnullになる問題を正規化する
			if m, ok := jsonData.(map[string]any); ok {
				normalizeNulls(m)
			}

			// 空文字列のenum値を除去する。Goのゼロ値で「未設定」を意味する
			if m, ok := jsonData.(map[string]any); ok {
				removeEmptyEnumValues(relaxed, m)
			}

			if err := relaxed.VisitJSON(jsonData); err != nil {
				errs = append(errs, fmt.Sprintf("%s[%d]: %v", e.schemaName, i, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("バリデーションエラー %d件:\n%s", len(errs), strings.Join(errs, "\n"))
	}

	return nil
}

// relaxSchema はスキーマからRequired制約を再帰的に除去したコピーを返す。
// $refで参照されるスキーマも解決してインライン化する
func relaxSchema(schema *openapi3.Schema, components openapi3.Schemas) *openapi3.Schema {
	if schema == nil {
		return nil
	}

	relaxed := *schema
	relaxed.Required = nil

	if schema.Properties != nil {
		relaxed.Properties = make(openapi3.Schemas, len(schema.Properties))
		for name, propRef := range schema.Properties {
			resolved := resolveSchemaRef(propRef, components)
			if resolved == nil {
				relaxed.Properties[name] = propRef
				continue
			}
			relaxedProp := relaxSchema(resolved, components)
			relaxed.Properties[name] = openapi3.NewSchemaRef("", relaxedProp)
		}
	}

	if schema.Items != nil {
		resolved := resolveSchemaRef(schema.Items, components)
		if resolved != nil {
			relaxedItems := relaxSchema(resolved, components)
			relaxed.Items = openapi3.NewSchemaRef("", relaxedItems)
		}
	}

	return &relaxed
}

// resolveSchemaRef は$refを解決してスキーマの実体を返す
func resolveSchemaRef(ref *openapi3.SchemaRef, components openapi3.Schemas) *openapi3.Schema {
	if ref == nil {
		return nil
	}
	if ref.Value != nil {
		return ref.Value
	}
	// $refからコンポーネント名を抽出して解決する
	if ref.Ref != "" && components != nil {
		name := ref.Ref[strings.LastIndex(ref.Ref, "/")+1:]
		if resolved, ok := components[name]; ok && resolved.Value != nil {
			return resolved.Value
		}
	}
	return nil
}

// removeEmptyEnumValues はenum定義を持つフィールドの空文字列を除去する。
// Goのゼロ値（空文字列）は「未設定」を意味するため、VisitJSONでenum違反にならないようにする
func removeEmptyEnumValues(schema *openapi3.Schema, data map[string]any) {
	if schema == nil || schema.Properties == nil {
		return
	}

	for propName, propRef := range schema.Properties {
		value, exists := data[propName]
		if !exists || value == nil {
			continue
		}

		propSchema := propRef.Value
		if propSchema == nil {
			continue
		}

		// enum定義を持つフィールドの空文字列を除去する
		if len(propSchema.Enum) > 0 {
			if str, ok := value.(string); ok && str == "" {
				delete(data, propName)
			}
			continue
		}

		// ネストされたオブジェクトを再帰的に処理する
		if nested, ok := value.(map[string]any); ok {
			removeEmptyEnumValues(propSchema, nested)
		}

		// 配列内のオブジェクトを処理する
		if propSchema.Items != nil && propSchema.Items.Value != nil {
			if arr, ok := value.([]any); ok {
				for _, elem := range arr {
					if nested, ok := elem.(map[string]any); ok {
						removeEmptyEnumValues(propSchema.Items.Value, nested)
					}
				}
			}
		}
	}
}

// normalizeNulls はGoのnilスライスがjson.Marshalでnullになる問題を解消する。
// nullの値を空配列に置き換え、ネストされたオブジェクトも再帰的に処理する
func normalizeNulls(data map[string]any) {
	for key, val := range data {
		if val == nil {
			data[key] = []any{}
			continue
		}
		if nested, ok := val.(map[string]any); ok {
			normalizeNulls(nested)
		}
	}
}

// toAnySlice は型付きスライスを[]anyに変換する
func toAnySlice[T any](items []T) []any {
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}
