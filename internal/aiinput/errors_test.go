package aiinput

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestAIError_WithEntity(t *testing.T) {
	t.Parallel()

	entity := ecs.Entity(42)
	err := &AIError{
		Type:    "planning",
		Message: "行動計画に失敗した",
		Entity:  &entity,
	}

	assert.Contains(t, err.Error(), "planning")
	assert.Contains(t, err.Error(), "Entity=42")
	assert.Contains(t, err.Error(), "行動計画に失敗した")
}

func TestAIError_WithoutEntity(t *testing.T) {
	t.Parallel()

	// Entity=nil は「エンティティ未設定」を表す
	err := &AIError{
		Type:    "vision",
		Message: "視界計算に失敗した",
		Entity:  nil,
	}

	assert.Contains(t, err.Error(), "vision")
	assert.Contains(t, err.Error(), "視界計算に失敗した")
	assert.NotContains(t, err.Error(), "Entity=")
}

func TestAIError_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := &AIError{Type: "test", Message: "test message"}
	assert.Contains(t, err.Error(), "test")
	assert.Contains(t, err.Error(), "test message")
}
