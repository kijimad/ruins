package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
		marginLeft   = 10.0 // 左マージン
		marginBottom = 10.0 // 下マージン
		badgeGap     = 4.0  // バッジ間の隙間
		paddingX     = 6.0  // バッジ内の左右パディング
		paddingY     = 4.0  // バッジ内の上下パディング
		maxVisible   = 5    // 最大表示数
	)

	// メッセージエリアの上に表示
	messageAreaHeight := float64(data.MessageAreaHeight)
	screenHeight := float64(data.ScreenDimensions.Height)
	baseY := screenHeight - messageAreaHeight - marginBottom

	// 表示するバッジを決定
	badges := data.Badges
	hasMore := false
	if len(badges) > maxVisible {
		badges = badges[:maxVisible]
		hasMore = true
	}

	// 下から上に向かって描画
	currentY := baseY
	for i := len(badges) - 1; i >= 0; i-- {
		badge := badges[i]

		// テキストサイズを測定
		textWidth, textHeight := text.Measure(badge.Text, sb.bodyFace, 0)

		// バッジの高さ
		badgeHeight := textHeight + paddingY*2

		// Y位置を計算（下から積み上げる）
		badgeY := currentY - badgeHeight

		// 背景矩形を描画
		bgX := float32(marginLeft - paddingX)
		bgWidth := float32(textWidth + paddingX*2)
		vector.FillRect(screen, bgX, float32(badgeY), bgWidth, float32(badgeHeight), badge.Color, false)

		// 白文字でテキストを描画
		textY := badgeY + paddingY
		drawOutlinedText(screen, badge.Text, sb.bodyFace, marginLeft, textY, color.White)

		// 次のバッジの位置を更新
		currentY = badgeY - badgeGap
	}

	// 表示しきれないバッジがある場合は「+N」を表示
	if hasMore {
		moreCount := len(data.Badges) - maxVisible
		moreText := fmt.Sprintf("+%d", moreCount)
		textWidth, textHeight := text.Measure(moreText, sb.bodyFace, 0)
		badgeHeight := textHeight + paddingY*2
		badgeY := currentY - badgeHeight

		// グレーの背景
		bgX := float32(marginLeft - paddingX)
		bgWidth := float32(textWidth + paddingX*2)
		vector.FillRect(screen, bgX, float32(badgeY), bgWidth, float32(badgeHeight), color.RGBA{100, 100, 100, 255}, false)

		textY := badgeY + paddingY
		drawOutlinedText(screen, moreText, sb.bodyFace, marginLeft, textY, color.White)
	}
}
