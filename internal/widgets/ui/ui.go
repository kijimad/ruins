// Package ui は画面層に公開する UI の構成面。プリミティブ実体 widgets/internal/uicore のうち、
// 構成に必要な最小の面だけを再輸出する。
//
// Atomic Design の page は atom のスタイルに触れない。その境界を Go の internal 可視性で
// API として強制する。塗り・枠・パディング・テクスチャ背景などの装飾は widgets 配下の
// 部品だけが実体パッケージを import して扱える。画面層からは装飾の型と関数がそもそも
// 見えないため、スタイルの手組みはコンパイルできない。
// Group は具象でなく Widget を返し、装飾メソッドを型の面からも隠す。
// 層の全体像は widgets/menuframe のパッケージコメントを参照。
package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	core "github.com/kijimaD/ruins/internal/widgets/internal/uicore"
)

// Widget は画面が受け取る UI ツリー。描くことだけができる。
//
// 配置は部品が済ませてから渡すので、画面には Layout の面を見せない。
// 見せると絶対座標で画面を組めてしまい、レイアウトエンジンを迂回する経路が型に開く。
// 部品はプリミティブ実体の Widget を扱い、そちらは Layout を持つ。
type Widget = core.Drawable

// Text は1行テキスト。画面が直接テキストを置くときに使う。
type Text = core.Text

// EbitenCanvas は ebiten の画面へ描く Canvas 実装。State の Draw から使う。
type EbitenCanvas = core.EbitenCanvas

// NewText はテキストウィジェットを作る。
func NewText(value string, face text.Face, c color.Color) *Text {
	return core.NewText(value, face, c)
}

// NewGroup は子を絶対配置で束ねる。ページ合成のルートに使う。
func NewGroup(children ...Widget) Widget { return core.NewGroup(core.Placeable(children)...) }

// NewEbitenCanvas は描画先スクリーンを与えて Canvas を作る。
func NewEbitenCanvas(screen *ebiten.Image) *EbitenCanvas { return core.NewEbitenCanvas(screen) }

// MeasureText は face で描いたときの s の送り幅と高さを画素で返す。
// 寸法を内容から決めたい箇所はこれを通す。text/v2 を直に呼ぶと丸め方が箇所ごとに割れる。
func MeasureText(s string, face text.Face) (int, int) { return core.MeasureText(s, face) }

// MeasureTextWidth は MeasureText の幅だけを返す。
func MeasureTextWidth(s string, face text.Face) int { return core.MeasureTextWidth(s, face) }
