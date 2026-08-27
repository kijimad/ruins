package messagewindow

import (
	"image"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	w "github.com/kijimaD/ruins/internal/world"
)

func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

// assertChoiceGolden は選択肢一覧を width×height の画面へ描いてゴールデンと比較する。
// renderChoiceList が返すツリーを EbitenCanvas で描く。窓の実描画と同じ経路を通す
func assertChoiceGolden(t *testing.T, config tabMenuConfig, state viewState, world w.World, width, height int) {
	t.Helper()
	vrt.AssertScreenGolden(t, func() func(*ebiten.Image) {
		// 元は自然幅のコンテナを左上へ描いていた。選択バーと区切り線はその幅いっぱいに伸びる
		cw := min(choiceBlockWidth(config, state, world), width)
		tree := renderChoiceList(config, state, world, image.Rect(0, 0, cw, height))
		return func(screen *ebiten.Image) {
			tree.Draw(ui.NewEbitenCanvas(screen))
		}
	}, width, height)
}

func TestGolden_SingleItem(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	assertChoiceGolden(t, tabMenuConfig{
		Tabs: []tabItem{
			{ID: "tab", Label: "タブ", Items: []item{
				{ID: "item1", Label: "アイテム1"},
			}},
		},
	}, viewState{TabIndex: 0, ItemIndex: 0}, world, 300, 50)
}

func TestGolden_MultipleItems_FirstSelected(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	assertChoiceGolden(t, tabMenuConfig{
		Tabs: []tabItem{
			{ID: "tab", Label: "タブ", Items: []item{
				{ID: "item1", Label: "回復薬"},
				{ID: "item2", Label: "鉄鉱石"},
				{ID: "item3", Label: "聖水"},
			}},
		},
	}, viewState{TabIndex: 0, ItemIndex: 0}, world, 300, 120)
}

func TestGolden_MultipleItems_MiddleSelected(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	assertChoiceGolden(t, tabMenuConfig{
		Tabs: []tabItem{
			{ID: "tab", Label: "タブ", Items: []item{
				{ID: "item1", Label: "回復薬"},
				{ID: "item2", Label: "鉄鉱石"},
				{ID: "item3", Label: "聖水"},
				{ID: "item4", Label: "毒消し"},
				{ID: "item5", Label: "火炎瓶"},
			}},
		},
	}, viewState{TabIndex: 0, ItemIndex: 2}, world, 300, 180)
}

func TestGolden_EmptyItems(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	assertChoiceGolden(t, tabMenuConfig{
		Tabs: []tabItem{
			{ID: "tab", Label: "タブ", Items: []item{}},
		},
	}, viewState{}, world, 300, 50)
}

func TestGolden_WithPagination(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	items := make([]item, 10)
	for i := range items {
		items[i] = item{
			ID:    "item",
			Label: "アイテム" + string(rune('A'+i)),
		}
	}
	assertChoiceGolden(t, tabMenuConfig{
		Tabs:         []tabItem{{ID: "tab", Label: "タブ", Items: items}},
		ItemsPerPage: 3,
	}, viewState{TabIndex: 0, ItemIndex: 0}, world, 300, 150)
}

func TestGolden_WithAdditionalLabels(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	assertChoiceGolden(t, tabMenuConfig{
		Tabs: []tabItem{
			{ID: "tab", Label: "タブ", Items: []item{
				{ID: "item1", Label: "回復薬", AdditionalLabels: []string{"x3", "1.5kg"}},
				{ID: "item2", Label: "鉄鉱石", AdditionalLabels: []string{"x12", "6.0kg"}},
				{ID: "item3", Label: "聖水", AdditionalLabels: []string{"x1", "0.5kg"}},
			}},
		},
	}, viewState{TabIndex: 0, ItemIndex: 0}, world, 400, 120)
}

func TestGolden_ManyItems_LastPage(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	items := make([]item, 8)
	for i := range items {
		items[i] = item{
			ID:    "item",
			Label: "アイテム" + string(rune('A'+i)),
		}
	}
	assertChoiceGolden(t, tabMenuConfig{
		Tabs:         []tabItem{{ID: "tab", Label: "タブ", Items: items}},
		ItemsPerPage: 3,
	}, viewState{TabIndex: 0, ItemIndex: 7}, world, 300, 120)
}
