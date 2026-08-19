// Package menuinput はメニュー操作のキーボード入力を Action へ変換し、再生時は world の
// 供給源へ差し替える。menuloop と overlay の両方から使い、入力層の差し替え点を1つにする。
// menuloop は overlay を import するため overlay から menuloop は呼べない。両者より下の
// このパッケージに置くことで、循環せず入力経路を共有できる。
//
// state 固有の追加キーは Binding の表で宣言し、変換の実行はこのパッケージが担う。
// キー読み取りが呼び出し側へ散らないので、キー→Action 対応をまとめて単体テストできる。
package menuinput

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

// Binding は state 固有の追加キーと Action の対応。共通キーより先に、表の順で評価する
type Binding struct {
	Key    ebiten.Key
	Shift  ShiftMode
	Action inputmapper.ActionID
}

// ReadMenuInput は1フレームぶんのメニュー操作を Action として読む。world が入力供給源を
// 持つならそこから読み、持たない本番ではキーボードから変換する。ここで入れ替わるのは
// キー入力の有無だけで、後段の DoAction 以降は本番と完全に同じ経路を通る。
// bindings は state 固有の追加キーで、共通キーより先に評価する
func ReadMenuInput(world w.World, bindings ...Binding) (inputmapper.ActionID, bool) {
	if src := world.Resources.MenuInput; src != nil {
		return src()
	}
	return convertKeys(input.GetSharedKeyboardInput(), bindings)
}

// convertKeys はキー入力を Action に変換する。本番の入力経路。
// 追加キーの束縛表を先に見てから、全メニュー共通のキーへ落ちる
func convertKeys(ki input.KeyboardInput, bindings []Binding) (inputmapper.ActionID, bool) {
	shift := ki.IsKeyPressed(ebiten.KeyShift)
	for _, b := range bindings {
		if !ki.IsKeyJustPressed(b.Key) {
			continue
		}
		if b.Shift == ShiftRequired && !shift {
			continue
		}
		if b.Shift == ShiftForbidden && shift {
			continue
		}
		return b.Action, true
	}

	if ki.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}
	// 左右キーはタブ切替に固定する。全メニューで意味を揃え、ページ送りには使わない。
	// 長い一覧は上下でカーソルがページ境界を越えると自動でページが繰られる
	if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) {
		return inputmapper.ActionMenuTabPrev, true
	}
	if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) {
		return inputmapper.ActionMenuTabNext, true
	}
	if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowUp) {
		return inputmapper.ActionMenuUp, true
	}
	if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowDown) {
		return inputmapper.ActionMenuDown, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyTab) {
		if shift {
			return inputmapper.ActionMenuTabPrev, true
		}
		return inputmapper.ActionMenuTabNext, true
	}
	if ki.IsEnterJustPressedOnce() {
		return inputmapper.ActionMenuSelect, true
	}
	return "", false
}
