package hud

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/widgets/framedbg"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
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
func (sb *StatusBadges) Draw(screen *ebiten.Image, data StatusBadgesData) {
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
		textWidth, textHeight := ui.MeasureText(badge.Text, sb.bodyFace)

		// バッジの高さ
		badgeHeight := float64(textHeight) + paddingY*2

		// Y位置を計算（下から積み上げる）
		badgeY := currentY - badgeHeight

		// 背景矩形を描画。塗りはバッジの状態色を保ちつつ、枠はメニュー枠と同じ共通 chrome に揃える
		bgX := theme.Space4
		bgWidth := textWidth + int(paddingX)*2
		framedbg.Draw(screen, bgX, int(badgeY), bgWidth, int(badgeHeight), badgeStyle(badge.Color))

		// 白文字でテキストを描画
		textY := badgeY + paddingY
		drawOutlinedText(screen, badge.Text, sb.bodyFace, theme.Space4F+paddingX, textY, theme.TextPrimary)

		// 次のバッジの位置を更新
		currentY = badgeY - badgeGap
	}

	// 表示しきれないバッジがある場合は「+N」を表示
	if hasMore {
		moreCount := len(data.Badges) - maxVisible
		moreText := fmt.Sprintf("+%d", moreCount)
		textWidth, textHeight := ui.MeasureText(moreText, sb.bodyFace)
		badgeHeight := float64(textHeight) + paddingY*2
		badgeY := currentY - badgeHeight

		// グレーの背景。他バッジと同じ共通 chrome に揃える
		bgX := theme.Space4
		bgWidth := textWidth + int(paddingX)*2
		framedbg.Draw(screen, bgX, int(badgeY), bgWidth, int(badgeHeight), badgeStyle(theme.HUDBadgeBg))

		textY := badgeY + paddingY
		drawOutlinedText(screen, moreText, sb.bodyFace, theme.Space4F, textY, theme.TextPrimary)
	}
}

// badgeStyle はバッジの共通 chrome スタイルを返す。塗りはバッジごとの状態色を渡し、
// 枠はメニュー枠と同じ PanelStyle に揃える
func badgeStyle(fill color.RGBA) framedbg.Style {
	s := framedbg.PanelStyle()
	s.BackgroundColor = fill
	return s
}
