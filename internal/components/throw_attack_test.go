package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestThrowAttack_GetElement(t *testing.T) {
	t.Parallel()

	assert.Equal(t, rawschema.ElementNONE, ThrowAttack{}.GetElement())
	assert.Equal(t, rawschema.ElementFIRE, ThrowAttack{Blast: &Blast{Element: rawschema.ElementFIRE}}.GetElement())
}
