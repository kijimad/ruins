package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPseudoCatalog_Translate(t *testing.T) {
	t.Parallel()
	c := NewPseudoCatalog()

	cases := []struct {
		name  string
		msgid string
		args  []any
		want  string
	}{
		{"英字を全角にする", "Start 1", nil, "Ｓｔａｒｔ 1"},
		{"書式の動詞を残して整形する", "%s took %d.", []any{"x", 3}, "x ｔｏｏｋ 3."},
		{"添字付きの動詞を残す", "%[2]s %[1]s", []any{"a", "b"}, "b a"},
		{"マークアップのタグを残す", "<item>Rock</item>", nil, "<item>Ｒｏｃｋ</item>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, c.Translate("ja", tc.msgid, tc.args...))
		})
	}
}
