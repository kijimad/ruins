package menuloop

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewKeyHelpState_渡した表を保持するファクトリを返す は生成される State が
// 渡した束縛表をそのまま保持することを固定する
func TestNewKeyHelpState_渡した表を保持するファクトリを返す(t *testing.T) {
	t.Parallel()
	table := []keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}}

	factory := NewKeyHelpState(table)
	state, err := factory()

	require.NoError(t, err)
	helpState, ok := state.(*KeyHelpState)
	require.True(t, ok)
	assert.Equal(t, table, helpState.table)
}

// TestHasEscapeLabel はラベル付き Escape 行の有無を判定することを固定する
func TestHasEscapeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		table []keybind.Binding
		want  bool
	}{
		{
			"Escapeにラベルがあればtrue",
			[]keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: "Back"}},
			true,
		},
		{
			"Escapeがラベル無しならfalse",
			[]keybind.Binding{{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel, Label: ""}},
			false,
		},
		{
			"Escapeが表に無ければfalse",
			[]keybind.Binding{{Key: ebiten.KeyEnter, Action: inputmapper.ActionMenuSelect, Label: "Confirm"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, hasEscapeLabel(tt.table))
		})
	}
}
