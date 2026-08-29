package messagelog_test

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
	"github.com/kijimaD/ruins/internal/widgets/messagelog"

	"github.com/kijimaD/ruins/internal/world/query"
)

func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestGolden_EmptyLog(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	vrt.AssertScreenGolden(t, func() func(*ebiten.Image) {
		w := messagelog.NewWidget(defaultConfig(), world)
		w.Update()
		return func(screen *ebiten.Image) {
			w.Draw(uicore.NewEbitenCanvas(screen), 0, 0, 400, 120)
		}
	}, 400, 120)
}

func TestGolden_SingleEntry(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	store := query.GetGameLog(world)
	store.Push("テストメッセージ")
	vrt.AssertScreenGolden(t, func() func(*ebiten.Image) {
		w := messagelog.NewWidget(defaultConfig(), world)
		w.Update()
		return func(screen *ebiten.Image) {
			w.Draw(uicore.NewEbitenCanvas(screen), 0, 0, 400, 120)
		}
	}, 400, 120)
}

func TestGolden_MultipleEntries(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	store := query.GetGameLog(world)
	store.Push("1行目のメッセージ")
	store.Push("2行目のメッセージ")
	store.Push("3行目のメッセージ")
	vrt.AssertScreenGolden(t, func() func(*ebiten.Image) {
		w := messagelog.NewWidget(defaultConfig(), world)
		w.Update()
		return func(screen *ebiten.Image) {
			w.Draw(uicore.NewEbitenCanvas(screen), 0, 0, 400, 120)
		}
	}, 400, 120)
}

func TestGolden_MaxLinesExceeded(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	store := query.GetGameLog(world)
	for range 10 {
		store.Push("メッセージ行")
	}
	vrt.AssertScreenGolden(t, func() func(*ebiten.Image) {
		w := messagelog.NewWidget(messagelog.WidgetConfig{
			MaxLines:   3,
			LineHeight: 20,
			Spacing:    2,
			Padding:    messagelog.Insets{Top: 4, Bottom: 4, Left: 8, Right: 8},
		}, world)
		w.Update()
		return func(screen *ebiten.Image) {
			w.Draw(uicore.NewEbitenCanvas(screen), 0, 0, 400, 80)
		}
	}, 400, 80)
}

func TestGolden_ColoredEntries(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	store := query.GetGameLog(world)
	gamelog.New(store).Markup(gamelog.Tag("error", "ダメージ")).Log()
	gamelog.New(store).Markup(gamelog.Tag("success", "回復")).Log()
	gamelog.New(store).Markup("通常" + gamelog.Tag("warning", "と") + gamelog.Tag("system", "混合")).Log()
	vrt.AssertScreenGolden(t, func() func(*ebiten.Image) {
		w := messagelog.NewWidget(defaultConfig(), world)
		w.Update()
		return func(screen *ebiten.Image) {
			w.Draw(uicore.NewEbitenCanvas(screen), 0, 0, 400, 120)
		}
	}, 400, 120)
}

func defaultConfig() messagelog.WidgetConfig {
	return messagelog.WidgetConfig{
		MaxLines:   5,
		LineHeight: 20,
		Spacing:    2,
		Padding:    messagelog.Insets{Top: 4, Bottom: 4, Left: 8, Right: 8},
	}
}
