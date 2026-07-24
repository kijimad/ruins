package messagewindow

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWindow_通常メッセージから内容を構築する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	msg := messagedata.NewDialogMessage("こんにちは", "案内人").
		WithChoice("はい", nil)

	win := NewWindow(world, msg)

	assert.True(t, win.IsOpen())
	assert.False(t, win.IsClosed())
	assert.Same(t, msg, win.CurrentMessage())
	assert.Equal(t, "案内人", win.content.SpeakerName)
	require.Len(t, win.content.Choices, 1)
	assert.Equal(t, "はい", win.content.Choices[0].Text)
	assert.False(t, win.queueManager.HasNext(), "連鎖メッセージが無ければキューは空")
}

func TestNewWindow_連鎖メッセージがある場合はキューに追加される(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	msg := messagedata.NewSystemMessage("最初").
		SystemMessage("次")

	win := NewWindow(world, msg)

	require.True(t, win.queueManager.HasNext())
	assert.Equal(t, 1, win.queueManager.Size())
}

func TestNewWindow_選択肢のActionが元のActionを呼びキューに追加する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actionCalled := false
	followUp := messagedata.NewSystemMessage("フォローアップ")
	msg := messagedata.NewDialogMessage("どうする？", "NPC").
		WithChoiceMessage("実行", followUp)
	msg.Choices[0].Action = func(_ w.World) error {
		actionCalled = true
		return nil
	}

	win := NewWindow(world, msg)
	require.Len(t, win.content.Choices, 1)

	err := win.content.Choices[0].Action()

	require.NoError(t, err)
	assert.True(t, actionCalled, "元のChoice.Actionが呼ばれる")
	require.True(t, win.queueManager.HasNext(), "MessageDataを伴う選択肢はキュー先頭に追加される")
	assert.Same(t, followUp, win.queueManager.Dequeue())
}

func TestNewWindow_選択肢にActionが無くてもエラーにならない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	msg := messagedata.NewDialogMessage("どうする？", "NPC").
		WithChoice("何もしない", nil)

	win := NewWindow(world, msg)

	err := win.content.Choices[0].Action()

	require.NoError(t, err)
}

func Test_hasMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines [][]messagedata.TextSegment
		want  bool
	}{
		{
			name:  "行が無い",
			lines: nil,
			want:  false,
		},
		{
			name:  "空白文字のみの行",
			lines: [][]messagedata.TextSegment{{{Text: "  \t\n"}}},
			want:  false,
		},
		{
			name:  "文字のある行",
			lines: [][]messagedata.TextSegment{{{Text: "こんにちは"}}},
			want:  true,
		},
		{
			name: "一部の行だけ文字がある",
			lines: [][]messagedata.TextSegment{
				{{Text: "   "}},
				{{Text: "本文"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			win := &Window{content: MessageContent{TextSegmentLines: tt.lines}}
			assert.Equal(t, tt.want, win.hasMessage())
		})
	}
}

func Test_calculateWindowSize(t *testing.T) {
	t.Parallel()

	t.Run("選択肢が無い場合はconfigの高さをそのまま使う", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := &Window{config: DefaultConfig(), world: world, hasChoices: false}

		size := win.calculateWindowSize()

		assert.Equal(t, WindowSize{Width: MinWidth, Height: MinHeight}, size)
	})

	t.Run("選択肢のみの場合は選択肢数から高さを算出する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := &Window{
			config:     DefaultConfig(),
			world:      world,
			hasChoices: true,
			content: MessageContent{
				Choices: []Choice{{Text: "選択肢1"}},
			},
		}

		size := win.calculateWindowSize()

		// メッセージ0 + 選択肢40*1 + top20 + bottom15 + タイトル0 + spacing0 = 75
		assert.Equal(t, 75, size.Height)
		assert.Equal(t, MinWidth, size.Width)
	})

	t.Run("メッセージと話者がある場合は高さに加算される", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := &Window{
			config:     DefaultConfig(),
			world:      world,
			hasChoices: true,
			content: MessageContent{
				SpeakerName:      "話者",
				TextSegmentLines: [][]messagedata.TextSegment{{{Text: "本文"}}},
				Choices:          []Choice{{Text: "選択肢1"}, {Text: "選択肢2"}},
			},
		}

		size := win.calculateWindowSize()

		// メッセージ150 + 選択肢40*2 + top20 + bottom15 + タイトル40 + spacing10 = 315
		assert.Equal(t, 315, size.Height)
	})

	t.Run("画面高さの80%を超える場合は上限で頭打ちになる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		choices := make([]Choice, 20)
		for i := range choices {
			choices[i] = Choice{Text: "選択肢"}
		}
		win := &Window{
			config:     DefaultConfig(),
			world:      world,
			hasChoices: true,
			content: MessageContent{
				SpeakerName:      "話者",
				TextSegmentLines: [][]messagedata.TextSegment{{{Text: "本文"}}},
				Choices:          choices,
			},
		}

		size := win.calculateWindowSize()

		assert.Equal(t, int(720*0.8), size.Height, "画面高さの80%である576に頭打ちになる")
	})
}

func Test_calculateWindowPosition(t *testing.T) {
	t.Parallel()

	t.Run("通常サイズは画面中央かつ上から1/4に配置される", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		x, y := win.calculateWindowPosition(WindowSize{Width: 600, Height: 300})

		assert.Equal(t, 180, x)
		assert.Equal(t, 180, y)
	})

	t.Run("下端をはみ出す場合は下マージンに合わせて引き上げる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		_, y := win.calculateWindowPosition(WindowSize{Width: 600, Height: 550})

		assert.Equal(t, 140, y)
	})

	t.Run("引き上げてもなお上マージンを割る場合は上マージンに固定する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		_, y := win.calculateWindowPosition(WindowSize{Width: 600, Height: 700})

		assert.Equal(t, 30, y, "上マージン30に固定される")
	})
}

func Test_calculateItemsPerPage(t *testing.T) {
	t.Parallel()

	t.Run("十分な余白があれば全件をそのまま返す", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		got := win.calculateItemsPerPage(5)

		assert.Equal(t, 5, got)
	})

	t.Run("メッセージや話者があると余白が狭まりページ数が減る", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{
			world: world,
			content: MessageContent{
				SpeakerName:      "話者",
				TextSegmentLines: [][]messagedata.TextSegment{{{Text: "本文"}}},
			},
		}

		got := win.calculateItemsPerPage(100)

		// 画面高720*0.8=576 から overhead 265 を引いた 311 を choiceItemHeight40 で割ると 7 になる。
		// overhead の内訳は message150 top20 bottom15 title40 spacing10 indicator30
		assert.Equal(t, 7, got)
	})

	t.Run("画面が小さいと最低3件は確保する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(200, 200)
		win := &Window{
			world: world,
			content: MessageContent{
				SpeakerName:      "話者",
				TextSegmentLines: [][]messagedata.TextSegment{{{Text: "本文"}}},
			},
		}

		got := win.calculateItemsPerPage(50)

		assert.Equal(t, 3, got)
	})

	t.Run("画面が大きくても最大15件に頭打ちになる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 10000)
		win := &Window{world: world}

		got := win.calculateItemsPerPage(50)

		assert.Equal(t, 15, got)
	})
}

func TestWindow_DoAction(t *testing.T) {
	t.Parallel()

	t.Run("ConfirmでWindowが閉じる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		win.DoAction(inputmapper.ActionConfirm)

		assert.True(t, win.IsClosed())
	})

	t.Run("Skipで閉じる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		win.DoAction(inputmapper.ActionSkip)

		assert.True(t, win.IsClosed())
	})

	t.Run("不正なActionはpanicする", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		assert.PanicsWithValue(t, "不正なアクション: unknown", func() {
			win.DoAction(inputmapper.ActionID("unknown"))
		})
	})
}

func TestWindow_Close(t *testing.T) {
	t.Parallel()

	t.Run("次のメッセージが無ければ閉じてonCloseが呼ばれる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))
		onCloseCalled := false
		win.onClose = func() { onCloseCalled = true }

		win.Close()

		assert.True(t, win.IsClosed())
		assert.True(t, onCloseCalled)
	})

	t.Run("次のメッセージがあれば閉じずに表示を切り替える", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		msg := messagedata.NewSystemMessage("最初").SystemMessage("次")
		win := NewWindow(world, msg)
		onCloseCalled := false
		win.onClose = func() { onCloseCalled = true }

		win.Close()

		assert.False(t, win.IsClosed(), "次のメッセージがある間は閉じない")
		assert.False(t, onCloseCalled)
		assert.Equal(t, msg.GetNextMessages()[0], win.CurrentMessage())
	})

	t.Run("OnCompleteが設定されていれば閉じる際に呼ばれる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		completeCalled := false
		msg := messagedata.NewSystemMessage("テスト").WithOnComplete(func() { completeCalled = true })
		win := NewWindow(world, msg)

		win.Close()

		assert.True(t, completeCalled)
	})

	t.Run("既に閉じている場合は何もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))
		win.Close()
		onCloseCalledCount := 0
		win.onClose = func() { onCloseCalledCount++ }

		win.Close()

		assert.Equal(t, 0, onCloseCalledCount, "二重に閉じてもonCloseは呼ばれない")
		assert.True(t, win.IsClosed(), "二重に閉じても閉じたまま")
	})
}
