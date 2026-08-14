package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserError_Error(t *testing.T) {
	t.Parallel()

	err := &UserError{Msg: "何も拾うものがない"}
	assert.Equal(t, "何も拾うものがない", err.Error())
}
