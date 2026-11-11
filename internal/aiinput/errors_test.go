package aiinput

import (
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAIError_WithEntity(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	err := &AIError{
		Type:    "planning",
		Message: "行動計画に失敗した",
		Entity:  &entity,
	}

	assert.Equal(t, fmt.Sprintf("AI Error [planning] Entity=%d: 行動計画に失敗した", entity), err.Error())
}

func TestAIError_WithoutEntity(t *testing.T) {
	t.Parallel()

	// Entity=nil は「エンティティ未設定」を表す
	err := &AIError{
		Type:    "vision",
		Message: "視界計算に失敗した",
		Entity:  nil,
	}

	assert.Equal(t, "AI Error [vision]: 視界計算に失敗した", err.Error())
}

func TestAIError_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := &AIError{Type: "test", Message: "test message"}
	assert.Equal(t, "AI Error [test]: test message", err.Error())
}
