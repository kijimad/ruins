package keybind

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// KeyToken はキー表記の1粒。Boxed はグリフ自体が白い箱を持つキーキャップ変種であることを示す。
// 箱を持たないトークンは描画側が背景の箱を敷き、全キーを白背景に黒グリフで揃える
type KeyToken struct {
	Text  string
	Boxed bool
}

// HintEntry はキー表記とラベル訳の組。ヒント行とキー一覧ヘルプが共有する表示単位。
// Keys は文字列文脈用の連結表記で、Tokens は箱の有無を保った粒のままの表記
type HintEntry struct {
	Keys   string
	Label  string
	Tokens []KeyToken
}

// HintEntries は束縛表から表示単位の一覧を導出する。表の順に並べ、Label が空の行は
// 隠しキーとして出さない。連続する同じ Label の行はキー表記を連結して1項目にまとめる。
// 隠しキーの行を挟んでも連続とみなすため、先に表示行だけへ絞ってからまとめる
func HintEntries(world w.World, table []Binding) []HintEntry {
	var labeled []Binding
	for _, b := range table {
		if b.Label != "" {
			labeled = append(labeled, b)
		}
	}

	var entries []HintEntry
	for i := 0; i < len(labeled); {
		// i から同じ Label が続く範囲が1項目。j が範囲の終端に進む
		var keys strings.Builder
		var tokens []KeyToken
		j := i
		for ; j < len(labeled) && labeled[j].Label == labeled[i].Label; j++ {
			keys.WriteString(KeyLabel(labeled[j]))
			tokens = append(tokens, KeyTokens(labeled[j])...)
		}
		entries = append(entries, HintEntry{Keys: keys.String(), Label: query.T(world, labeled[i].Label), Tokens: tokens})
		i = j
	}
	return entries
}

// NavHint は束縛表からキー操作ヒントの1行を組む。表を変えれば表示が追随し、
// 挙動とヒントの二重管理をなくす。戻る操作は常に最後に置くため、
// ActionMenuCancel と ActionCloseMenu の行だけ末尾へ回す
func NavHint(world w.World, table []Binding) string {
	var rows []Binding
	var closes []Binding
	for _, b := range table {
		if b.Action == inputmapper.ActionMenuCancel || b.Action == inputmapper.ActionCloseMenu {
			closes = append(closes, b)
			continue
		}
		rows = append(rows, b)
	}
	parts := make([]string, 0, 8)
	for _, e := range HintEntries(world, append(rows, closes...)) {
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

// KeyTokens は Binding のキー表記をトークン列で返す。Shift 併用は Shift 記号のトークンを前置する。
// 箱入りのキーキャップグリフは Boxed を立て、描画側が二重に箱を敷かないようにする
func KeyTokens(b Binding) []KeyToken {
	if b.Key == ebiten.KeySlash && b.Shift == ShiftRequired {
		return []KeyToken{{Text: consts.IconKeyHelp, Boxed: true}}
	}
	// 数字キーはキーキャップグリフで表す
	if b.Key >= ebiten.KeyDigit0 && b.Key <= ebiten.KeyDigit9 {
		return []KeyToken{{Text: consts.IconKeyDigitBoxes[b.Key-ebiten.KeyDigit0], Boxed: true}}
	}
	// 英字キーはキーキャップグリフで表す。グリフは大文字デザインなので、
	// Shift 併用は大文字化でなく Shift 記号の前置で表す
	if b.Key >= ebiten.KeyA && b.Key <= ebiten.KeyZ {
		keycap := KeyToken{Text: string(consts.IconKeyAlphaBoxBase + rune(b.Key-ebiten.KeyA)), Boxed: true}
		if b.Shift == ShiftRequired {
			return []KeyToken{{Text: consts.IconKeyShift}, keycap}
		}
		return []KeyToken{keycap}
	}
	switch b.Key {
	case ebiten.KeyArrowLeft:
		return []KeyToken{{Text: consts.IconKeyArrowLeft, Boxed: true}}
	case ebiten.KeyArrowRight:
		return []KeyToken{{Text: consts.IconKeyArrowRight, Boxed: true}}
	case ebiten.KeyArrowUp:
		return []KeyToken{{Text: consts.IconKeyArrowUp, Boxed: true}}
	case ebiten.KeyArrowDown:
		return []KeyToken{{Text: consts.IconKeyArrowDown, Boxed: true}}
	case ebiten.KeyEnter:
		return []KeyToken{{Text: consts.IconKeyEnter}}
	case ebiten.KeyEscape:
		return []KeyToken{{Text: consts.IconKeyEsc}}
	case ebiten.KeySpace:
		return []KeyToken{{Text: consts.IconKeySpace}}
	case ebiten.KeyTab:
		return []KeyToken{{Text: consts.IconKeyTab}}
	case ebiten.KeyPeriod:
		return []KeyToken{{Text: consts.IconKeyDot}}
	default:
		// 専用表記の無いキーは内部名をそのまま出す
		return []KeyToken{{Text: b.Key.String()}}
	}
}

// KeyLabel は Binding のキー表記を文字列文脈用に連結して返す。フッターなど
// 装飾を敷けない場所で使う。表記の規約は KeyTokens が持つ
func KeyLabel(b Binding) string {
	var sb strings.Builder
	for _, tok := range KeyTokens(b) {
		sb.WriteString(tok.Text)
	}
	return sb.String()
}
