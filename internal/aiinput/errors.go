package aiinput

import (
	"fmt"

	"github.com/mlange-42/ark/ecs"
)

// AIError はAI処理に関するエラーを表す。
// Entity はnilで「エンティティ未設定」を表す。
type AIError struct {
	Type    string      // エラーの種類
	Message string      // エラーメッセージ
	Entity  *ecs.Entity // 関連するエンティティ。nilは未設定を表す
}

// Error はerrorインターフェースを実装する
func (e *AIError) Error() string {
	if e.Entity != nil {
		return fmt.Sprintf("AI Error [%s] Entity=%d: %s", e.Type, *e.Entity, e.Message)
	}
	return fmt.Sprintf("AI Error [%s]: %s", e.Type, e.Message)
}
