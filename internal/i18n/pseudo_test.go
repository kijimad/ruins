package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalog_Translate_疑似訳(t *testing.T) {
	t.Parallel()
	c := NewCatalog()

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
		{"動詞にならない % の後の英字も写す", "100%, done", nil, "100%, ｄｏｎｅ"},
		{"末尾の % を残す", "100%", nil, "100%"},
		{"幅と精度付きの動詞と %% を残す", "%5.1f%% hit", []any{12.5}, " 12.5% ｈｉｔ"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, c.Translate(PseudoLang, Msgid(tc.msgid), tc.args...))
		})
	}
}
