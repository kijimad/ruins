package menuscreen

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	w "github.com/kijimaD/ruins/internal/world"
)

// Overlay はメニュー本体に重ねる小窓の共通契約。詳細モーダルやアクション窓が満たす。
// Active なら入力を専有し、閉じるまで下位の overlay とメニュー本体の操作を止める。
// Screen は登録順を優先順位として、Active な最上位だけに入力を渡し、Active なものを重ねて描く
type Overlay interface {
	// Active は overlay を表示中かを返す
	Active() bool
	// HandleInput は表示中のキー入力を処理する。UI を作り直すべきなら第1返り値を true にする
	HandleInput(world w.World) (dirty bool, err error)
	// Window は overlay のウィンドウを rect の位置へ組み立てる。対象が無ければ nil を返す
	Window(world w.World, rect image.Rectangle) *widget.Window
}

var (
	_ Overlay = (*Detail)(nil)
	_ Overlay = (*ActionWindow)(nil)
)
