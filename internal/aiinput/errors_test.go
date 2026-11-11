package aiinput

import (
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.ErrorContains(t, err, "planning")
	require.ErrorContains(t, err, fmt.Sprintf("Entity=%d", entity))
	require.ErrorContains(t, err, "行動計画に失敗した")
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
	require.ErrorContains(t, err, "test")
	require.ErrorContains(t, err, "test message")
}
