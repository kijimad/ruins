package keybind

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// HintEntry はキー表記とラベル訳の組。ヒント行とキー一覧ヘルプが共有する表示単位。
// Keys は文字列文脈用の連結表記で、Tokens は粒のままの表記
type HintEntry struct {
	Keys   string
	Label  string
	Tokens []string
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
		var tokens []string
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
// 表記は全てアイコンフォントの箱なしグリフで、本文フォントの文字は混ぜない。
// フォント合成の倍率差でサイズが割れるのを防ぐ。キーキャップの箱は描画側が一律に敷く
func KeyTokens(b Binding) []string {
	if b.Key == ebiten.KeySlash && b.Shift == ShiftRequired {
		return []string{consts.IconQuestion}
	}
	// 数字キーは数字グリフで表す
	if b.Key >= ebiten.KeyDigit0 && b.Key <= ebiten.KeyDigit9 {
		return []string{string(consts.IconKeyDigitBase + rune(b.Key-ebiten.KeyDigit0))}
	}
	// 英字キーは英字グリフで表す。Shift 併用は Shift 記号の前置で表す
	if b.Key >= ebiten.KeyA && b.Key <= ebiten.KeyZ {
		keycap := string(consts.IconKeyAlphaBase + rune(b.Key-ebiten.KeyA))
		if b.Shift == ShiftRequired {
			return []string{consts.IconKeyShift, keycap}
		}
		return []string{keycap}
	}
	switch b.Key {
	case ebiten.KeyArrowLeft:
		return []string{consts.IconArrowLeft}
	case ebiten.KeyArrowRight:
		return []string{consts.IconArrowRight}
	case ebiten.KeyArrowUp:
		return []string{consts.IconArrowUp}
	case ebiten.KeyArrowDown:
		return []string{consts.IconArrowDown}
	case ebiten.KeyEnter:
		return []string{consts.IconKeyEnter}
	case ebiten.KeyEscape:
		return []string{consts.IconKeyEsc}
	case ebiten.KeySpace:
		return []string{consts.IconKeySpace}
	case ebiten.KeyTab:
		return []string{consts.IconKeyTab}
	case ebiten.KeyPeriod:
		return []string{consts.IconKeyDot}
	default:
		// 専用表記の無いキーは内部名をそのまま出す
		return []string{b.Key.String()}
	}
}

// KeyLabel は Binding のキー表記を文字列文脈用に連結して返す。フッターなど
// 装飾を敷けない場所で使う。表記の規約は KeyTokens が持つが、? だけは ASCII で出す。
// 文字列文脈は本文フォントと混ざるため、倍率の違うアイコングリフではサイズが割れる
func KeyLabel(b Binding) string {
	if b.Key == ebiten.KeySlash && b.Shift == ShiftRequired {
		return "?"
	}
	return strings.Join(KeyTokens(b), "")
}
