package save

import (
	"encoding/json"
	"fmt"

	"github.com/kijimaD/ruins/internal/oapi"
)

const saveDataSchemaName = "SaveData.SaveData"

// ValidateSaveJSON はセーブデータのJSON文字列をOpenAPIスキーマでバリデーションする
func ValidateSaveJSON(jsonData string) error {
	spec, err := oapi.GetSpec()
	if err != nil {
		return fmt.Errorf("OpenAPIスキーマの読み込みに失敗: %w", err)
	}
	schemaRef, ok := spec.Components.Schemas[saveDataSchemaName]
	if !ok {
		return fmt.Errorf("OpenAPIスキーマに%sコンポーネントが見つからない", saveDataSchemaName)
	}
	if schemaRef.Value == nil {
		return fmt.Errorf("%sスキーマの値がnil", saveDataSchemaName)
	}

	var jsonObj any
	if err := json.Unmarshal([]byte(jsonData), &jsonObj); err != nil {
		return fmt.Errorf("JSONパースに失敗: %w", err)
	}

	if err := schemaRef.Value.VisitJSON(jsonObj); err != nil {
		return fmt.Errorf("セーブデータのバリデーションエラー: %w", err)
	}

	return nil
}
