package states

import (
	"testing"
	"unicode/utf8"

	"github.com/ebitenui/ebitenui/widget"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNameValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "1文字の名前は有効",
			input:    "A",
			expected: true,
		},
		{
			name:     "10文字の名前は有効",
			input:    "ABCDEFGHIJ",
			expected: true,
		},
		{
			name:     "空文字は無効",
			input:    "",
			expected: false,
		},
		{
			name:     "11文字の名前は無効",
			input:    "ABCDEFGHIJK",
			expected: false,
		},
		{
			name:     "日本語1文字は有効",
			input:    "あ",
			expected: true,
		},
		{
			name:     "日本語10文字は有効",
			input:    "あいうえおかきくけこ",
			expected: true,
		},
		{
			name:     "日本語11文字は無効",
			input:    "あいうえおかきくけこさ",
			expected: false,
		},
		{
			name:     "混合文字で10文字は有効",
			input:    "Ash太郎1234",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nameLen := utf8.RuneCountInString(tt.input)
			isValid := nameLen >= nameMinLength && nameLen <= nameMaxLength
			assert.Equal(t, tt.expected, isValid)
		})
	}
}

func TestConfirmName_ChangesPlayerName(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.textInput = widget.NewTextInput()
	st.errorText = widget.NewText()

	st.textInput.SetText("NewName")
	st.confirmName(world)

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "NewName", nameComp.Name)
}

func TestConfirmName_Japanese(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.textInput = widget.NewTextInput()
	st.errorText = widget.NewText()

	st.textInput.SetText("太郎")
	st.confirmName(world)

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "太郎", nameComp.Name)
}

func TestConfirmName_InvalidLength(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.textInput = widget.NewTextInput()
	st.errorText = widget.NewText()

	// 空文字は無効
	st.textInput.SetText("")
	st.confirmName(world)
	assert.NotEmpty(t, st.errorText.Label)

	// 名前は変更されていない
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "Ash", nameComp.Name)
}

func TestConfirmName_TooLong(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.textInput = widget.NewTextInput()
	st.errorText = widget.NewText()

	// 11文字は無効
	st.textInput.SetText("ABCDEFGHIJK")
	st.confirmName(world)
	assert.NotEmpty(t, st.errorText.Label)

	// 名前は変更されていない
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "Ash", nameComp.Name)
}
