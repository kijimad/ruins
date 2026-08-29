package overlay

import (
	"image"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// Layer はメニュー本体に重ねる小窓の共通契約。詳細モーダルやアクション窓が満たす。
// Active なら入力を専有し、閉じるまで下位の overlay とメニュー本体の操作を止める。
// Screen は登録順を優先順位として、Active な最上位だけに入力を渡し、Active なものを重ねて描く。
// 描画の手段は overlay ごとに WindowRenderer か ScreenRenderer のどちらかで表す。
type Layer interface {
	// Active は overlay を表示中かを返す
	Active() bool
	// HandleInput は表示中のキー入力を処理する。Screen は毎フレーム再構築するので dirty は返さない
	HandleInput(world w.World) error
}

// ScreenRenderer は自身を internal/uicore のツリーとして描く overlay。Screen は本体を描いたあと、
// このツリーを画面の上へ重ねて描く。ツリーは rect に配置済みで返す。表示するものが無ければ nil を返す。
type ScreenRenderer interface {
	RenderOverlay(world w.World, rect image.Rectangle) uicore.Drawable
}

var _ Layer = (*Detail)(nil)
