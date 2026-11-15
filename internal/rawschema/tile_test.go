package rawschema_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestTile_IsBox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		box  *bool
		want bool
	}{
		{"欄を省くと箱でない", nil, false},
		{"false なら箱でない", new(false), false},
		{"true なら箱", new(true), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, rawschema.Tile{Box: tt.box}.IsBox())
		})
	}
}
