package messagelog

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/ui"
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

// Widget はメッセージログ表示ウィジェット。
//
// 各エントリを1行とし、行内の色付きフラグメントを水平に連ねて描く。描画は internal/ui の
// ツリーで組み、グローバル可変状態に触れない。フラグメント幅はフェイスの測定で決める
type Widget struct {
	config      WidgetConfig
	world       w.World
	entries     []gamelog.LogEntry
	lastVersion int
	loaded      bool
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
	widget.refresh()
}

// Draw はウィジェットを指定位置に描画する
func (widget *Widget) Draw(screen *ebiten.Image, x, y, width, height int) {
	// モーダルなど別 state が前面にある間、このウィジェットを所有する state は Update されない。
	// それでも最新ログを映すため、描画時にも取り込みを行う。version 一致時は即座に戻るため軽量
	widget.refresh()

	if width <= 0 || height <= 0 {
		return
	}

	offscreen := ebiten.NewImage(width, height)
	widget.buildTree(width, height).Draw(ui.NewEbitenCanvas(offscreen))

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(offscreen, op)
}

// refresh はログのバージョンが変わったときだけ最新エントリを取り込む
func (widget *Widget) refresh() {
	store := query.GetGameLog(widget.world)
	if store == nil {
		return
	}
	version := store.Version()
	if widget.loaded && version == widget.lastVersion {
		return
	}
	widget.entries = store.GetRecentEntries(widget.config.MaxLines)
	widget.lastVersion = version
	widget.loaded = true
}

// buildTree はエントリ列を internal/ui のツリーへ組む。
// 各行を Padding.Top から LineHeight+Spacing 刻みで下へ、行内フラグメントは測定幅ぶん右へ並べる
func (widget *Widget) buildTree(width, height int) ui.Widget {
	res := widget.world.Resources.UIResources
	face := res.Text.BodyFace

	var children []ui.Widget
	x0 := widget.config.Padding.Left
	y := widget.config.Padding.Top

	visible := 0
	for _, entry := range widget.entries {
		if entry.IsEmpty() {
			continue
		}
		x := x0
		for _, fragment := range entry.Fragments {
			if fragment.Text == "" {
				continue
			}
			t := ui.NewText(fragment.Text, face, fragment.Color)
			t.Layout(image.Rect(x, y, width, y+widget.config.LineHeight))
			children = append(children, t)
			adv, _ := text.Measure(fragment.Text, face, 0)
			x += int(adv)
		}
		visible++
		y += widget.config.LineHeight + widget.config.Spacing
	}

	// エントリが無いときは案内文を出す
	if visible == 0 {
		t := ui.NewText(query.T(widget.world, "No log messages"), face, theme.TextSecondary)
		t.Layout(image.Rect(x0, widget.config.Padding.Top, width, widget.config.Padding.Top+widget.config.LineHeight))
		children = append(children, t)
	}

	root := ui.NewGroup(children...)
	root.Layout(image.Rect(0, 0, width, height))
	return root
}
