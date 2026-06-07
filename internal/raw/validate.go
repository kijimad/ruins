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
