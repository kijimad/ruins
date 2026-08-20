package keybind

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// HintEntry はキー表記とラベル訳の組。ヒント行とキー一覧ヘルプが共有する表示単位
type HintEntry struct {
	Keys  string
	Label string
}

// HintEntries は束縛表から表示単位の一覧を導出する。表の順に並べ、Label が空の行は
// 隠しキーとして出さない。連続する同じ Label の行はキー表記を連結して1項目にまとめる
func HintEntries(world w.World, tables ...[]Binding) []HintEntry {
	var entries []HintEntry
	label := ""
	keys := ""
	flush := func() {
		if label != "" {
			entries = append(entries, HintEntry{Keys: keys, Label: query.T(world, label)})
		}
		label, keys = "", ""
	}
	for _, table := range tables {
		for _, b := range table {
			if b.Label == "" {
				continue
			}
			if b.Label != label {
				flush()
				label = b.Label
			}
			keys += KeyLabel(b)
		}
		flush()
	}
	return entries
}

// NavHint は束縛表からキー操作ヒントの1行を組む。表を変えれば表示が追随し、
// 挙動とヒントの二重管理をなくす。戻る操作は常に最後に置くため、
// ActionMenuCancel と ActionCloseMenu の行だけ末尾へ回す
func NavHint(world w.World, tables ...[]Binding) string {
	var rows []Binding
	var closes []Binding
	for _, table := range tables {
		for _, b := range table {
			if b.Action == inputmapper.ActionMenuCancel || b.Action == inputmapper.ActionCloseMenu {
				closes = append(closes, b)
				continue
			}
			rows = append(rows, b)
		}
	}
	parts := make([]string, 0, 8)
	for _, e := range HintEntries(world, rows, closes) {
		parts = append(parts, e.Keys+" "+e.Label)
	}
	return strings.Join(parts, "   ")
}

// HelpHint はキー一覧ヘルプへの入口だけを示すフッター文字列を組む。
// 各画面の全キー列挙はヘルプ画面が担い、常設のフッターは入口の1項目に絞って画面を静かに保つ
func HelpHint(world w.World) string {
	var rows []Binding
	for _, b := range MenuCommon {
		if b.Action == inputmapper.ActionOpenKeyHelp {
			rows = append(rows, b)
		}
	}
	return NavHint(world, rows)
}

// KeyLabel は Binding のキー表記を返す。矢印や Enter/Esc は素の記号がフォントに無く
// 文字化けするため FontAwesome のアイコンを使う。文字キーは小文字で表し、
// Shift 併用は大文字で表す。verbList の KeyHint と同じ表記規約
func KeyLabel(b Binding) string {
	if b.Key == ebiten.KeySlash && b.Shift == ShiftRequired {
		return "?"
	}
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
