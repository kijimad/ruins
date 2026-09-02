package hud

import (
	"fmt"
	"image"
	"image/color"
	"slices"

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

// Draw はステータスバッジを描画する
func (sb *StatusBadges) Draw(cv uicore.Canvas, data StatusBadgesData) {
	if !sb.enabled || len(data.Badges) == 0 {
		return
	}

	const (
		badgeGap   = 4.0 // バッジ間の隙間
		paddingX   = 6.0 // バッジ内の左右パディング
		paddingY   = 4.0 // バッジ内の上下パディング
		maxVisible = 5   // 最大表示数
	)

	// メッセージエリアの上に表示
	messageAreaHeight := float64(data.MessageAreaHeight)
	screenHeight := float64(data.ScreenDimensions.Height)
	baseY := screenHeight - messageAreaHeight - theme.Space4F

	// 表示するバッジを決定
	badges := data.Badges
	hasMore := false
	if len(badges) > maxVisible {
		badges = badges[:maxVisible]
		hasMore = true
	}

	// 下から上に向かって描画
	currentY := baseY
	for _, badge := range slices.Backward(badges) {
		// テキストサイズを測定
		textWidth, textHeight := uicore.MeasureText(badge.Text, sb.bodyFace)

		// バッジの高さ
		badgeHeight := float64(textHeight) + paddingY*2

		// Y位置を計算（下から積み上げる）
		badgeY := currentY - badgeHeight

		// 背景矩形を描画。塗りはバッジの状態色を保ちつつ、枠はメニュー枠と同じ共通 chrome に揃える
		bgX := theme.Space4
		bgWidth := uicore.FitWidth([]int{textWidth}, int(paddingX)*2, 0)
		badgeChrome(cv, image.Rect(bgX, int(badgeY), bgX+bgWidth, int(badgeY+badgeHeight)), badge.Color)

		// 白文字でテキストを描画
		textY := badgeY + paddingY
		drawOutlinedText(cv, badge.Text, sb.bodyFace, image.Pt(int(theme.Space4F+paddingX), int(textY)), theme.TextPrimary)

		// 次のバッジの位置を更新
		currentY = badgeY - badgeGap
	}

	// 表示しきれないバッジがある場合は「+N」を表示
	if hasMore {
		moreCount := len(data.Badges) - maxVisible
		moreText := fmt.Sprintf("+%d", moreCount)
		textWidth, textHeight := uicore.MeasureText(moreText, sb.bodyFace)
		badgeHeight := float64(textHeight) + paddingY*2
		badgeY := currentY - badgeHeight

		// グレーの背景。他バッジと同じ共通 chrome に揃える
		bgX := theme.Space4
		bgWidth := uicore.FitWidth([]int{textWidth}, int(paddingX)*2, 0)
		badgeChrome(cv, image.Rect(bgX, int(badgeY), bgX+bgWidth, int(badgeY+badgeHeight)), theme.HUDBadgeBg)

		textY := badgeY + paddingY
		drawOutlinedText(cv, moreText, sb.bodyFace, image.Pt(int(theme.Space4F+paddingX), int(textY)), theme.TextPrimary)
	}
}

// badgeChrome はバッジの箱を描く。塗りの色は状態を表すデータなので、意匠のテクスチャでなく
// 塗りと枠で表す。枠の色はパネルと同じにして、HUD の他の箱と質感を揃える
func badgeChrome(cv uicore.Canvas, r image.Rectangle, fill color.RGBA) {
	cv.StrokeRect(r, 1, theme.PanelHighlight)
	inner := image.Rect(r.Min.X+1, r.Min.Y+1, r.Max.X-1, r.Max.Y-1)
	cv.FillRect(inner, fill)
	cv.FillRect(image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, inner.Min.Y+1), theme.PanelHighlight)
	cv.FillRect(image.Rect(inner.Min.X, inner.Max.Y-1, inner.Max.X, inner.Max.Y), theme.PanelShadow)
}
