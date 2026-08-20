// Package keybind はキーボード入力を Action へ変換し、再生時は world の供給源へ差し替える。
// メニュー・overlay・フィールドの全文脈から使い、入力層の差し替え点を1つにする。
// menuloop は overlay を import するため overlay から menuloop は呼べない。両者より下の
// このパッケージに置くことで、循環せず入力経路を共有できる。
//
// 文脈ごとのキーは Binding の表で宣言し、変換の実行はこのパッケージが担う。
// キー読み取りが呼び出し側へ散らないので、キー→Action 対応をまとめて単体テストできる。
package keybind

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
)

// ShiftMode は Binding が Shift 押下をどう扱うかを表す
type ShiftMode int

const (
	// ShiftAny は Shift の押下状態を問わない
	ShiftAny ShiftMode = iota
	// ShiftRequired は Shift が押されているときだけ一致する
	ShiftRequired
	// ShiftForbidden は Shift が押されていないときだけ一致する
	ShiftForbidden
)

// PressMode はキーの押下判定の種類
type PressMode int

const (
	// PressJust は押した瞬間の1フレームだけ発火する
	PressJust PressMode = iota
	// PressRepeat は押しっぱなしでリピート発火する。移動やカーソル送りに使う
	PressRepeat
)

// Binding は1つのキー操作と Action の対応。表の順で評価する
type Binding struct {
	Key    ebiten.Key
	Shift  ShiftMode
	Press  PressMode
	Held   *ebiten.Key // 同時押しが要る追加キー。nil なら単独キーで一致する。斜め移動のような2キー同時押しで使う
	Action inputmapper.ActionID
	Label  string // ヒント表示の msgid。空なら隠しキーとしてヒントに出さない
}

// MenuCommon はメニュー共通のキー束縛。各メニューの固有表の後に渡して評価する。
// 左右キーはタブ切替に固定する。全メニューで意味を揃え、ページ送りには使わない。
// 長い一覧は上下でカーソルがページ境界を越えると自動でページが繰られる。
// Shift+Tab の行を Tab の行より先に置き、Shift 併用を先に判定する。
// Tab キーは左右の別名なので Label を持たせずヒントに出さない
var MenuCommon = []Binding{
	{Key: ebiten.KeyArrowLeft, Press: PressRepeat, Action: inputmapper.ActionMenuTabPrev, Label: "Tab"},
	{Key: ebiten.KeyArrowRight, Press: PressRepeat, Action: inputmapper.ActionMenuTabNext, Label: "Tab"},
	{Key: ebiten.KeyArrowUp, Press: PressRepeat, Action: inputmapper.ActionMenuUp, Label: "Select"},
	{Key: ebiten.KeyArrowDown, Press: PressRepeat, Action: inputmapper.ActionMenuDown, Label: "Select"},
	{Key: ebiten.KeyTab, Shift: ShiftRequired, Press: PressJust, Action: inputmapper.ActionMenuTabPrev},
	{Key: ebiten.KeyTab, Press: PressJust, Action: inputmapper.ActionMenuTabNext},
	{Key: ebiten.KeyEnter, Press: PressJust, Action: inputmapper.ActionMenuSelect, Label: "Confirm"},
	{Key: ebiten.KeySlash, Shift: ShiftRequired, Press: PressJust, Action: inputmapper.ActionOpenKeyHelp, Label: "Help"},
	{Key: ebiten.KeyEscape, Press: PressJust, Action: inputmapper.ActionMenuCancel, Label: "Back"},
}

// ReadInput は1フレームぶんの入力を Action として読む。world が入力供給源を
// 持つならそこから読み、持たない本番では tables を表の順に評価してキーボードから変換する。
// ここで入れ替わるのはキー入力の有無だけで、後段の DoAction 以降は本番と完全に同じ経路を通る
func ReadInput(world w.World, tables ...[]Binding) (inputmapper.ActionID, bool) {
	if src := world.Resources.InputSource; src != nil {
		return src()
	}
	return Convert(input.GetSharedKeyboardInput(), tables...)
}

// Convert はキー入力を Action に変換する。本番の入力経路。
// tables を先頭から順に評価し、最初に一致した行の Action を返す。
// Shift と同時押しの条件を押下判定より先に見て、リピート判定の副作用を条件外の行で起こさない
func Convert(ki input.KeyboardInput, tables ...[]Binding) (inputmapper.ActionID, bool) {
	shift := ki.IsKeyPressed(ebiten.KeyShift)
	for _, table := range tables {
		for _, b := range table {
			if b.Shift == ShiftRequired && !shift {
				continue
			}
			if b.Shift == ShiftForbidden && shift {
				continue
			}
			if b.Held != nil && !ki.IsKeyPressed(*b.Held) {
				continue
			}
			if !pressed(ki, b) {
				continue
			}
			return b.Action, true
		}
	}
	return "", false
}

// pressed は Binding の押下モードに応じたキー判定を返す。
// Enter は押下押上のワンセット検出というデバイス層の癖を持つため、ここでだけ特別に扱う
func pressed(ki input.KeyboardInput, b Binding) bool {
	if b.Press == PressRepeat {
		return ki.IsKeyPressedWithRepeat(b.Key)
	}
	if b.Key == ebiten.KeyEnter {
		return ki.IsEnterJustPressedOnce()
	}
	return ki.IsKeyJustPressed(b.Key)
}
