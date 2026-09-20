package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUserSettings(t *testing.T) {
	t.Parallel()

	// コンストラクタが言語コードを格納することだけ確かめる。値の写経を避け1本にする
	s := NewUserSettings("ja")
	assert.Equal(t, "ja", s.Language)
}
