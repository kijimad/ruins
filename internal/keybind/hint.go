package keybind

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// NavHint は束縛表からキー操作ヒントの1行を組む。表を変えれば表示が追随し、
// 挙動とヒントの二重管理をなくす。tables を表の順に並べ、Label が空の行は隠しキーとして出さない。
// 連続する同じ Label の行はキー表記を連結して1項目にまとめる。戻る操作は常に最後に置くため、
// ActionMenuCancel の行だけ末尾へ回す。hasTabs が偽ならタブ切替の行を出さない
func NavHint(world w.World, hasTabs bool, tables ...[]Binding) string {
	var parts []string
	var cancels []string
	label := ""
	keys := ""
	flush := func() {
		if label != "" {
			parts = append(parts, keys+" "+query.T(world, label))
		}
		label, keys = "", ""
	}
	for _, table := range tables {
		for _, b := range table {
			if b.Label == "" {
				continue
			}
			if !hasTabs && (b.Action == inputmapper.ActionMenuTabPrev || b.Action == inputmapper.ActionMenuTabNext) {
				continue
			}
			if b.Action == inputmapper.ActionMenuCancel {
				cancels = append(cancels, keyLabel(b)+" "+query.T(world, b.Label))
				continue
			}
			if b.Label != label {
				flush()
				label = b.Label
			}
			keys += keyLabel(b)
		}
		flush()
	}
	parts = append(parts, cancels...)
	return strings.Join(parts, "   ")
}

// keyLabel は Binding のキー表記を返す。矢印や Enter/Esc は素の記号がフォントに無く
// 文字化けするため FontAwesome のアイコンを使う。文字キーは小文字で表し、
// Shift 併用は大文字で表す。verbList の KeyHint と同じ表記規約
func keyLabel(b Binding) string {
	switch b.Key {
	case ebiten.KeyArrowLeft:
		return consts.IconArrowLeft
	case ebiten.KeyArrowRight:
		return consts.IconArrowRight
	case ebiten.KeyArrowUp:
		return consts.IconArrowUp
	case ebiten.KeyArrowDown:
		return consts.IconArrowDown
	case ebiten.KeyEnter:
		return consts.IconKeyEnter
	case ebiten.KeyEscape:
		return consts.IconKeyEsc
	default:
		name := b.Key.String()
		if b.Shift == ShiftRequired {
			return strings.ToUpper(name)
		}
		return strings.ToLower(name)
	}
}
