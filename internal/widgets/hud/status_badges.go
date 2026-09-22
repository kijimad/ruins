package hud

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// StatusBadge はステータスバッジの情報
type StatusBadge struct {
	Text  string     // 表示テキスト
	Color color.RGBA // 背景色
}

// StatusBadgesData はステータスバッジ表示に必要なデータ
type StatusBadgesData struct {
	Badges            []StatusBadge    // 表示するバッジ一覧
	MessageAreaHeight int              // メッセージエリアの高さ
	ScreenDimensions  ScreenDimensions // 画面サイズ
}

// StatusBadges は左下にステータスバッジを表示するウィジェット
type StatusBadges struct {
	bodyFace text.Face
	enabled  bool
}

// NewStatusBadges は新しい StatusBadges を作成する
func NewStatusBadges(bodyFace text.Face) *StatusBadges {
	return &StatusBadges{
		bodyFace: bodyFace,
		enabled:  true,
	}
}

// Draw はステータスバッジを左下へ下から積んで描く。メッセージエリアの上端を底にして、下寄せの
// FlexColumn へ各バッジを流す。画面下端からの逆算や段の手積みをしない。表示上限を超えた分は先頭の
// 「+N」バッジにまとめる。
func (sb *StatusBadges) Draw(cv uicore.Canvas, data StatusBadgesData) {
	if !sb.enabled || len(data.Badges) == 0 {
		return
	}

	const (
		badgeGap   = 4 // バッジ間の隙間
		paddingX   = 6 // バッジ内の左右パディング
		paddingY   = 4 // バッジ内の上下パディング
		maxVisible = 5 // 最大表示数
	)

	badges := data.Badges
	hasMore := false
	if len(badges) > maxVisible {
		badges = badges[:maxVisible]
		hasMore = true
	}

	_, textHeight := uicore.MeasureText("W", sb.bodyFace)
	badgeHeight := textHeight + paddingY*2

	items := []uicore.FlexItem{{Grow: true}}
	add := func(str string, fill color.RGBA) {
		items = append(items,
			uicore.FlexItem{W: &badgeWidget{text: str, face: sb.bodyFace, fill: fill, padX: paddingX, padY: paddingY}, Height: badgeHeight},
			uicore.FlexItem{Height: badgeGap},
		)
	}

	// 上から順に積む。上限超過の「+N」を最上段、続けて表示するバッジを並べる。下寄せなので最後の
	// バッジがメッセージエリアの直上へ来る
	if hasMore {
		add(fmt.Sprintf("+%d", len(data.Badges)-maxVisible), theme.HUDBadgeBg)
	}
	for _, badge := range badges {
		add(badge.Text, badge.Color)
	}
	// 末尾の隙間をメッセージエリア分の下マージンへ置き換え、スタックの底をログ枠の上へ固定する
	items[len(items)-1] = uicore.FlexItem{Height: data.MessageAreaHeight + theme.Space4}

	inner := image.Rect(theme.Space4, 0, data.ScreenDimensions.Width, data.ScreenDimensions.Height)
	uicore.FlexColumn(inner, items)
	drawFlexItems(cv, items)
}

// badgeWidget は1つのステータスバッジ。与えられた矩形の左端から内容幅ぶんの箱を描き、白文字を重ねる。
// FlexColumn は幅を内側いっぱいへ伸ばすが、バッジは内容幅の左寄せなので Draw で左端から必要幅だけ描く。
type badgeWidget struct {
	rect       image.Rectangle
	text       string
	face       text.Face
	fill       color.RGBA
	padX, padY int
}

// Layout は uicore.Widget を満たす。
func (b *badgeWidget) Layout(r image.Rectangle) { b.rect = r }

// Draw は uicore.Widget を満たす。左端から内容幅の箱を描き、白文字を padding ぶん内側へ置く。
func (b *badgeWidget) Draw(cv uicore.Canvas) {
	textWidth, _ := uicore.MeasureText(b.text, b.face)
	bgWidth := uicore.FitWidth([]int{textWidth}, b.padX*2, 0)
	box := image.Rect(b.rect.Min.X, b.rect.Min.Y, b.rect.Min.X+bgWidth, b.rect.Max.Y)
	badgeChrome(cv, box, b.fill)
	drawOutlinedText(cv, b.text, b.face, image.Pt(b.rect.Min.X+b.padX, b.rect.Min.Y+b.padY), theme.TextPrimary)
}

// Children は uicore.Widget を満たす。子は持たない。
func (b *badgeWidget) Children() []uicore.Widget { return nil }

// badgeChrome はバッジの箱を角丸で描く。塗りの色は状態を表すデータなので、意匠のテクスチャでなく
// 塗りと枠で表す。枠の色はパネルと同じにして、HUD の他の箱と質感を揃える
func badgeChrome(cv uicore.Canvas, r image.Rectangle, fill color.RGBA) {
	cv.FillRect(r, fill, theme.CornerRadius)
	cv.StrokeRect(r, 1, theme.PanelHighlight, theme.CornerRadius)
}
