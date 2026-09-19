package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUserSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang string
	}{
		{"ja"},
		{"en"},
		{""},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()
			s := NewUserSettings(tt.lang)
			assert.Equal(t, tt.lang, s.Language)
		})
	}
}
