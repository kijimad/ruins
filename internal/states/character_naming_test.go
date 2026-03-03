package states

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
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

func TestNameLengthConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, nameMinLength, "最小文字数は1")
	assert.Equal(t, 10, nameMaxLength, "最大文字数は10")
}
