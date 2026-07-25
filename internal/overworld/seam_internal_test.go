package overworld

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAutotileBase は SpriteKey からオートタイルの基底名を切り出す判定を固定する。数値サフィックスが
// 無い記号はオートタイル対象外で、継ぎ目再計算がスキップする境界になる。
func TestAutotileBase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key      string
		wantBase string
		wantOK   bool
	}{
		{"dirt_15", "dirt", true},       // 通常のオートタイル
		{"dirt_0", "dirt", true},        // 端スプライト
		{"grass_wall_3", "grass_wall", true}, // 最後の _ で切るので基底に _ を含んでよい
		{"void", "", false},             // アンダースコア無しは非オートタイル
		{"dirt_x", "", false},           // 数値でないサフィックス
		{"dirt_", "", false},            // 空サフィックス
	}
	for _, c := range cases {
		base, ok := autotileBase(c.key)
		assert.Equalf(t, c.wantOK, ok, "autotileBase(%q) の ok", c.key)
		if c.wantOK {
			assert.Equalf(t, c.wantBase, base, "autotileBase(%q) の base", c.key)
		}
	}
}
