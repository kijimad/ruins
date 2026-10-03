package messagewindow

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_buildTree(t *testing.T) {
	t.Parallel()

	t.Run("話者も本文も選択肢も無ければEnterプロンプトだけを組む", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := &Window{config: defaultWindowConfig(), world: world}

		tree := win.buildTree()

		require.Len(t, tree.Children(), 3, "枠・選択バー・Enter文字の3つ")
		assert.Equal(t, []string{"Enter"}, uicore.CollectLabels(tree))
	})

	t.Run("話者がある場合はタイトルバーと名前を追加する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := &Window{
			config:  defaultWindowConfig(),
			world:   world,
			content: messageContent{SpeakerName: "案内人"},
		}

		tree := win.buildTree()

		require.Len(t, tree.Children(), 5, "枠・タイトルバー・名前・選択バー・Enter文字の5つ")
		assert.Equal(t, []string{"案内人", "Enter"}, uicore.CollectLabels(tree))
	})

	t.Run("本文がある場合は本文のテキストを追加する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := &Window{
			config: defaultWindowConfig(),
			world:  world,
			content: messageContent{
				TextSegmentLines: [][]messagedata.TextSegment{{{Text: "本文"}}},
			},
		}

		tree := win.buildTree()

		assert.Equal(t, []string{"本文", "Enter"}, uicore.CollectLabels(tree))
	})

	t.Run("選択肢がある場合はEnterプロンプトの代わりに選択肢一覧を組む", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := &Window{
			config:     defaultWindowConfig(),
			world:      world,
			hasChoices: true,
			choiceConfig: tabMenuConfig{
				Tabs: []tabItem{{ID: "choices", Items: []item{
					{ID: "a", Label: "はい"},
					{ID: "b", Label: "いいえ"},
				}}},
			},
		}

		tree := win.buildTree()

		require.Len(t, tree.Children(), 2, "枠と選択肢一覧の2つ")
		labels := uicore.CollectLabels(tree)
		assert.Contains(t, labels, "はい")
		assert.Contains(t, labels, "いいえ")
		assert.NotContains(t, labels, "Enter", "選択肢があるときはEnterプロンプトを出さない")
	})
}

func Test_segmentedLineWidgets(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	face := world.Resources.UIResources.Text.BodyFace

	t.Run("空白のみの行は何も描かず行高だけ進める", func(t *testing.T) {
		t.Parallel()

		win := &Window{
			config:  defaultWindowConfig(),
			content: messageContent{TextSegmentLines: [][]messagedata.TextSegment{{{Text: "  \t"}}}},
		}

		out := win.segmentedLineWidgets(0, 100, 0, face)

		assert.Empty(t, out)
	})

	t.Run("文字がある行はデフォルト色のTextを1つ返す", func(t *testing.T) {
		t.Parallel()

		win := &Window{
			config:  defaultWindowConfig(),
			content: messageContent{TextSegmentLines: [][]messagedata.TextSegment{{{Text: "こんにちは"}}}},
		}

		out := win.segmentedLineWidgets(0, 100, 0, face)

		require.Len(t, out, 1)
		txt, ok := out[0].(*uicore.Text)
		require.True(t, ok)
		assert.Equal(t, "こんにちは", txt.Value)
		assert.Equal(t, win.config.textStyle.Color, txt.Color)
	})

	t.Run("セグメントにColorがあればそれを使う", func(t *testing.T) {
		t.Parallel()

		custom := color.RGBA{R: 200, G: 10, B: 10, A: 255}
		win := &Window{
			config:  defaultWindowConfig(),
			content: messageContent{TextSegmentLines: [][]messagedata.TextSegment{{{Text: "警告", Color: &custom}}}},
		}

		out := win.segmentedLineWidgets(0, 100, 0, face)

		require.Len(t, out, 1)
		txt, ok := out[0].(*uicore.Text)
		require.True(t, ok)
		assert.Equal(t, custom, txt.Color)
	})

	t.Run("背景色付きセグメントは本文の前にPanelを追加する", func(t *testing.T) {
		t.Parallel()

		bg := color.RGBA{R: 50, G: 50, B: 50, A: 255}
		win := &Window{
			config:  defaultWindowConfig(),
			content: messageContent{TextSegmentLines: [][]messagedata.TextSegment{{{Text: "注目", BackgroundColor: &bg}}}},
		}

		out := win.segmentedLineWidgets(0, 100, 0, face)

		require.Len(t, out, 2, "背景Panelと本文Textの2つ")
		_, isContainer := out[0].(*uicore.Container)
		assert.True(t, isContainer, "1つ目は背景のPanel")
		txt, ok := out[1].(*uicore.Text)
		require.True(t, ok)
		assert.Equal(t, "注目", txt.Value)
	})

	t.Run("空文字のセグメントはスキップする", func(t *testing.T) {
		t.Parallel()

		win := &Window{
			config: defaultWindowConfig(),
			content: messageContent{TextSegmentLines: [][]messagedata.TextSegment{
				{{Text: ""}, {Text: "あり"}},
			}},
		}

		out := win.segmentedLineWidgets(0, 100, 0, face)

		require.Len(t, out, 1)
		txt, ok := out[0].(*uicore.Text)
		require.True(t, ok)
		assert.Equal(t, "あり", txt.Value)
	})
}
