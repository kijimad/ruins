package rawschema_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestSpriteRender_HasShape_形が一致するかを返す(t *testing.T) {
	t.Parallel()

	facing := rawschema.Facing
	assert.True(t, rawschema.SpriteRender{Shape: &facing}.HasShape(rawschema.Facing))
	assert.False(t, rawschema.SpriteRender{Shape: &facing}.HasShape(rawschema.Inset))
	assert.False(t, rawschema.SpriteRender{}.HasShape(rawschema.Facing), "形を省いた立て板")
}
