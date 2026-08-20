// Package keybind はキーボード入力を Action へ変換し、再生時は world の供給源へ差し替える。
// メニュー・overlay・フィールドの全文脈から使い、入力層の差し替え点を1つにする。
// menuloop は overlay を import するため overlay から menuloop は呼べない。両者より下の
// このパッケージに置くことで、循環せず入力経路を共有できる。
//
// 文脈ごとのキーは Binding の表で宣言し、変換の実行はこのパッケージが担う。
// キー読み取りが呼び出し側へ散らないので、キー→Action 対応をまとめて単体テストできる。
package keybind

import (
	"fmt"

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

// MenuCommon はメニュー共通のキー束縛。画面固有の断片と MustMerge で1枚に合成して使う。
// 左右キーはタブ切替に固定する。全メニューで意味を揃え、ページ送りには使わない。
// 長い一覧は上下でカーソルがページ境界を越えると自動でページが繰られる。
// 全行の条件が互いに素なので、行の順序に意味は無い。
// Tab キーは左右の別名なので Label を持たせずヒントに出さない
var MenuCommon = []Binding{
	{Key: ebiten.KeyArrowLeft, Press: PressRepeat, Action: inputmapper.ActionMenuTabPrev, Label: "Tab"},
	{Key: ebiten.KeyArrowRight, Press: PressRepeat, Action: inputmapper.ActionMenuTabNext, Label: "Tab"},
	{Key: ebiten.KeyArrowUp, Press: PressRepeat, Action: inputmapper.ActionMenuUp, Label: "Select"},
	{Key: ebiten.KeyArrowDown, Press: PressRepeat, Action: inputmapper.ActionMenuDown, Label: "Select"},
	{Key: ebiten.KeyTab, Shift: ShiftRequired, Press: PressJust, Action: inputmapper.ActionMenuTabPrev},
	{Key: ebiten.KeyTab, Shift: ShiftForbidden, Press: PressJust, Action: inputmapper.ActionMenuTabNext},
	{Key: ebiten.KeyEnter, Press: PressJust, Action: inputmapper.ActionMenuSelect, Label: "Confirm"},
	{Key: ebiten.KeySlash, Shift: ShiftRequired, Press: PressJust, Action: inputmapper.ActionOpenKeyHelp, Label: "Help"},
	{Key: ebiten.KeyEscape, Press: PressJust, Action: inputmapper.ActionMenuCancel, Label: "Back"},
}

// ReadInput は1フレームぶんの入力を Action として読む。world が入力供給源を
// 持つならそこから読み、持たない本番ではキーボードから変換する。
// ここで入れ替わるのはキー入力の有無だけで、後段の DoAction 以降は本番と完全に同じ経路を通る。
// table は MustMerge 済みの1枚を渡す。実行時に表を重ねる階層は持たない
func ReadInput(world w.World, table []Binding) (inputmapper.ActionID, bool) {
	if src := world.Resources.InputSource; src != nil {
		return src()
	}
	return Convert(input.GetSharedKeyboardInput(), table)
}

// Convert はキー入力を Action に変換する。本番の入力経路。
// 表の行は条件が互いに素になるよう MustMerge が検証済みなので、評価順に意味は無い。
// 唯一の例外は同キー同 Shift で Held だけ違う同時押しの対で、両方成立する縮退入力だけ先の行が勝つ。
// Shift と同時押しの条件を押下判定より先に見て、リピート判定の副作用を条件外の行で起こさない
func Convert(ki input.KeyboardInput, table []Binding) (inputmapper.ActionID, bool) {
	shift := ki.IsKeyPressed(ebiten.KeyShift)
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
	return "", false
}

// MustMerge は束縛表の断片を1枚に合成する。条件が重なる行の組があれば panic で拒否し、
// ある断片が別の断片のキーを黙って影で食う継承的な事故を構築時に締め出す。
// 全パッケージ変数の合成は起動と単体テストの両方で必ず通るため、事実上コンパイル時検査になる
func MustMerge(fragments ...[]Binding) []Binding {
	var table []Binding
	for _, f := range fragments {
		table = append(table, f...)
	}
	if err := validate(table); err != nil {
		panic(err)
	}
	return table
}

// validate は表の全行対で条件の重なりを検査する。同じキーで Shift 条件が交差し、
// かつ同時押し条件も交差する組は、どちらが発火するかが行順に依存するため拒否する。
// 同キー同 Shift でも Held が互いに異なる非 nil の対は、斜め移動のような対等の同時押し仲間として許す
func validate(table []Binding) error {
	for i, a := range table {
		for _, b := range table[i+1:] {
			if a.Key != b.Key {
				continue
			}
			shiftOverlap := a.Shift == ShiftAny || b.Shift == ShiftAny || a.Shift == b.Shift
			if !shiftOverlap {
				continue
			}
			heldOverlap := a.Held == nil || b.Held == nil || *a.Held == *b.Held
			if !heldOverlap {
				continue
			}
			return fmt.Errorf("keybind: overlapping bindings: %s and %s compete for key %v", a.Action, b.Action, a.Key)
		}
	}
	return nil
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
