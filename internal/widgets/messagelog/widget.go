package messagelog

import (
	"github.com/ebitenui/ebitenui"
	euiwidget "github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
)

// Insets はパディング設定を表す
type Insets struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

// WidgetConfig はMessageLogWidgetの設定を表す
type WidgetConfig struct {
	MaxLines   int    // 表示する最大行数
	LineHeight int    // 1行の高さ
	Spacing    int    // 行間のスペース
	Padding    Insets // 内部パディング
}

// Widget はメッセージログ表示ウィジェット
type Widget struct {
	ui        *ebitenui.UI
	lastCount int
	config    WidgetConfig
	world     w.World
}

// NewWidget は新しいMessageLogWidgetを作成する
func NewWidget(config WidgetConfig, world w.World) *Widget {
	return &Widget{
		config: config,
		world:  world,
	}
}

// Update はウィジェットを更新する
func (widget *Widget) Update() {
	if widget.ui == nil {
		widget.initUI()
		return
	}

	// ログメッセージが更新されている場合はUIを再構築
	widget.updateUI()

	// UIを更新
	widget.ui.Update()
}

// Draw はウィジェットを指定位置に描画する
func (widget *Widget) Draw(screen *ebiten.Image, x, y, width, height int) {
	if widget.ui == nil {
		return
	}

	// オフスクリーン作成
	if width > 0 && height > 0 {
		offscreen := ebiten.NewImage(width, height)
		widget.ui.Draw(offscreen)

		// 描画位置を調整
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(offscreen, op)
	}
}

// initUI は初期UIを作成する
func (widget *Widget) initUI() {
	store := query.GetGameLog(widget.world)
	if store == nil {
		return
	}

	// 色付きログエントリを取得
	entries := store.GetRecentEntries(widget.config.MaxLines)

	// 色付きログエントリ用のコンテナを作成
	logContainer := widget.createColoredLogContainer(entries)

	// UIを初期化
	widget.ui = &ebitenui.UI{Container: logContainer}

	// 初期バージョンを設定
	widget.lastCount = store.Version()
}

// updateUI はログメッセージが更新された場合にUIを再構築する
func (widget *Widget) updateUI() {
	store := query.GetGameLog(widget.world)
	if store == nil {
		return
	}

	currentVersion := store.Version()

	// バージョンが変わっていない場合は更新不要
	if currentVersion == widget.lastCount {
		return
	}

	// 色付きログエントリを取得
	entries := store.GetRecentEntries(widget.config.MaxLines)

	// 色付きログエントリ用のコンテナを作成
	logContainer := widget.createColoredLogContainer(entries)

	// UIを更新
	widget.ui.Container = logContainer

	// バージョンを更新
	widget.lastCount = currentVersion
}

// createColoredLogContainer は色付きログエントリ用のコンテナを作成
func (widget *Widget) createColoredLogContainer(entries []gamelog.LogEntry) *euiwidget.Container {
	// ログ用コンテナを作成（縦並び）
	logContainer := euiwidget.NewContainer(
		euiwidget.ContainerOpts.Layout(
			euiwidget.NewRowLayout(
				euiwidget.RowLayoutOpts.Direction(euiwidget.DirectionVertical),
				euiwidget.RowLayoutOpts.Spacing(widget.config.Spacing),
				euiwidget.RowLayoutOpts.Padding(&euiwidget.Insets{
					Top:    widget.config.Padding.Top,
					Bottom: widget.config.Padding.Bottom,
					Left:   widget.config.Padding.Left,
					Right:  widget.config.Padding.Right,
				}),
			),
		),
	)

	// 各エントリを処理
	for _, entry := range entries {
		if entry.IsEmpty() {
			continue
		}

		// エントリ内の複数フラグメントを水平に並べるコンテナ
		entryContainer := euiwidget.NewContainer(
			euiwidget.ContainerOpts.Layout(
				euiwidget.NewRowLayout(
					euiwidget.RowLayoutOpts.Direction(euiwidget.DirectionHorizontal),
					euiwidget.RowLayoutOpts.Spacing(0),                   // フラグメント間のスペースなし
					euiwidget.RowLayoutOpts.Padding(&euiwidget.Insets{}), // パディングなし
				),
			),
			euiwidget.ContainerOpts.WidgetOpts(
				euiwidget.WidgetOpts.LayoutData(euiwidget.RowLayoutData{
					Stretch: false, // コンテナ自体も伸ばさない
				}),
			),
		)

		// 各フラグメントを色付きテキストとして追加
		for _, fragment := range entry.Fragments {
			if fragment.Text == "" {
				continue
			}

			// 文字数分だけのサイズのフラグメント専用テキストを使用
			fragmentWidget := styled.NewFragmentText(
				fragment.Text,
				fragment.Color, // フラグメント固有の色を使用
				widget.world.Resources.UIResources,
			)
			entryContainer.AddChild(fragmentWidget)
		}

		logContainer.AddChild(entryContainer)
	}

	// エントリがない場合
	if len(entries) == 0 {
		placeholderWidget := styled.NewFragmentText("ログメッセージなし", theme.TextSecondary, widget.world.Resources.UIResources)
		logContainer.AddChild(placeholderWidget)
	}

	return logContainer
}
