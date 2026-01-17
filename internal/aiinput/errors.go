package aiinput

import (
	"fmt"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// AIError はAI処理に関するエラーを表す
type AIError struct {
	Type    string     // エラーの種類
	Message string     // エラーメッセージ
	Entity  ecs.Entity // 関連するエンティティ
}

// Error はerrorインターフェースを実装する
func (e *AIError) Error() string {
	if e.Entity != 0 {
		return fmt.Sprintf("AI Error [%s] Entity=%d: %s", e.Type, e.Entity, e.Message)
	}
	return fmt.Sprintf("AI Error [%s]: %s", e.Type, e.Message)
}
