// Package ui は画面層に公開する UI の構成面。プリミティブ実体 widgets/internal/ui のうち、
// 構成に必要な最小の面だけを再輸出する。
//
// Atomic Design の page は atom のスタイルに触れない。その境界を Go の internal 可視性で
// API として強制する。塗り・枠・パディング・テクスチャ背景などの装飾は widgets 配下の
// 部品だけが実体パッケージを import して扱える。画面層からは装飾の型と関数がそもそも
// 見えないため、スタイルの手組みはコンパイルできない。
// Row・VBox・Group は具象でなく Widget を返し、装飾メソッドを型の面からも隠す。
// 層の全体像は widgets/menuframe のパッケージコメントを参照。
package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	core "github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// Widget は画面が受け取る UI ツリー。描くことだけができる。
//
// 配置は部品が済ませてから渡すので、画面には Layout の面を見せない。
// 見せると絶対座標で画面を組めてしまい、レイアウトエンジンを迂回する経路が型に開く。
// 部品はプリミティブ実体の Widget を扱い、そちらは Layout を持つ。
type Widget = core.Drawable

// Canvas は描画先。画面は EbitenCanvas を作ってツリーへ渡す。
type Canvas = core.Canvas

// Text は1行テキスト。Align・VCenter は内容のそろえで、画面が自由に指定してよい。
type Text = core.Text

// Align はテキストのそろえ方向。
type Align = core.Align

// そろえ方向の定数。
const (
	AlignLeft   = core.AlignLeft
	AlignRight  = core.AlignRight
	AlignCenter = core.AlignCenter
)

// EbitenCanvas は ebiten の画面へ描く Canvas 実装。State の Draw から使う。
type EbitenCanvas = core.EbitenCanvas

// NewText はテキストウィジェットを作る。
func NewText(value string, face text.Face, c color.Color) *Text {
	return core.NewText(value, face, c)
}

// NewGraphic は画像ウィジェットを作る。
func NewGraphic(img *ebiten.Image) Widget { return core.NewGraphic(img) }

// NewGroup は子を絶対配置で束ねる。ページ合成のルートに使う。
func NewGroup(children ...Widget) Widget { return core.NewGroup(core.Placeable(children)...) }

// VBox は子を縦に積む。rowH は各行の高さ。
func VBox(rowH int, children ...Widget) Widget { return core.VBox(rowH, core.Placeable(children)...) }

// Row は子を横に並べる。colWidths は各列の幅。0 の列は余り幅を吸って伸びる。
func Row(colWidths []int, cells ...Widget) Widget {
	return core.Row(colWidths, core.Placeable(cells)...)
}

// NewEbitenCanvas は描画先スクリーンを与えて Canvas を作る。
func NewEbitenCanvas(screen *ebiten.Image) *EbitenCanvas { return core.NewEbitenCanvas(screen) }

// MeasureText は face で描いたときの s の送り幅と高さを画素で返す。
// 寸法を内容から決めたい箇所はこれを通す。text/v2 を直に呼ぶと丸め方が箇所ごとに割れる。
func MeasureText(s string, face text.Face) (int, int) { return core.MeasureText(s, face) }

// MeasureTextWidth は MeasureText の幅だけを返す。
func MeasureTextWidth(s string, face text.Face) int { return core.MeasureTextWidth(s, face) }

// LineHeight は face の自然な行送りを画素で返す。行送りを固定値で持たずに済ませる。
func LineHeight(face text.Face) int { return core.LineHeight(face) }
