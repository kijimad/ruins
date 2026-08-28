package messagewindow

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/theme"
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

			win := &Window{content: messageContent{TextSegmentLines: tt.lines}}
			assert.Equal(t, tt.want, win.hasMessage())
		})
	}
}

func Test_calculateWindowSize(t *testing.T) {
	t.Parallel()

	t.Run("選択肢が無い場合はconfigの高さをそのまま使う", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := &Window{config: defaultWindowConfig(), world: world, hasChoices: false}

		size := win.calculateWindowSize()

		assert.Equal(t, windowSize{Width: MinWidth, Height: MinHeight}, size)
	})

	t.Run("選択肢のみの場合は選択肢数から高さを算出する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := &Window{
			config:     defaultWindowConfig(),
			world:      world,
			hasChoices: true,
			content: messageContent{
				Choices: []choiceOption{{Text: "選択肢1"}},
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
			config:     defaultWindowConfig(),
			world:      world,
			hasChoices: true,
			content: messageContent{
				SpeakerName:      "話者",
				TextSegmentLines: [][]messagedata.TextSegment{{{Text: "本文"}}},
				Choices:          []choiceOption{{Text: "選択肢1"}, {Text: "選択肢2"}},
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
		choices := make([]choiceOption, 20)
		for i := range choices {
			choices[i] = choiceOption{Text: "選択肢"}
		}
		win := &Window{
			config:     defaultWindowConfig(),
			world:      world,
			hasChoices: true,
			content: messageContent{
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

	t.Run("通常サイズは画面中央かつ上端はMenuWindowTopに揃う", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		x, y := win.calculateWindowPosition(windowSize{Width: 600, Height: 300})

		assert.Equal(t, 180, x)
		assert.Equal(t, theme.MenuWindowTop, y)
	})

	t.Run("下端をはみ出す場合は下マージンに合わせて引き上げる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		_, y := win.calculateWindowPosition(windowSize{Width: 600, Height: 660})

		assert.Equal(t, 30, y)
	})

	t.Run("引き上げてもなお上マージンを割る場合は上マージンに固定する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(960, 720)
		win := &Window{world: world}

		_, y := win.calculateWindowPosition(windowSize{Width: 600, Height: 700})

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
			content: messageContent{
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
			content: messageContent{
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

		assert.PanicsWithValue(t, "invalid action: unknown", func() {
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

func Test_showNextMessage(t *testing.T) {
	t.Parallel()

	t.Run("queueManagerが未設定なら何もしない", func(t *testing.T) {
		t.Parallel()

		win := &Window{isOpen: true}

		win.showNextMessage()

		assert.True(t, win.IsOpen(), "queueManagerが無いので状態は変化しない")
	})

	t.Run("キューが空ならウィンドウを閉じてonCloseを呼ぶ", func(t *testing.T) {
		t.Parallel()

		onCloseCalled := false
		win := &Window{
			isOpen:       true,
			queueManager: newQueueManager(),
			onClose:      func() { onCloseCalled = true },
		}

		win.showNextMessage()

		assert.True(t, win.IsClosed())
		assert.True(t, onCloseCalled)
	})
}

func Test_updateContentFromMessage(t *testing.T) {
	t.Parallel()

	t.Run("Actionがエラーを返すとMessageDataはキューに追加されない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		wantErr := errors.New("action失敗")
		followUp := messagedata.NewSystemMessage("フォローアップ")
		msg := messagedata.NewDialogMessage("どうする？", "NPC").
			WithChoiceMessage("実行", followUp)
		msg.Choices[0].Action = func(_ w.World) error { return wantErr }

		win := NewWindow(world, msg)
		err := win.content.Choices[0].Action()

		require.ErrorIs(t, err, wantErr)
		assert.False(t, win.queueManager.HasNext(), "Actionがエラーの場合はMessageDataが追加されない")
	})

	t.Run("queueManager未設定でMessageDataがある選択を実行すると生成して追加する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		followUp := messagedata.NewSystemMessage("フォローアップ")
		msg := messagedata.NewDialogMessage("どうする？", "NPC").
			WithChoiceMessage("実行", followUp)

		win := &Window{world: world}
		win.updateContentFromMessage(msg)
		require.Nil(t, win.queueManager, "この時点ではqueueManagerは未設定")

		err := win.content.Choices[0].Action()

		require.NoError(t, err)
		require.NotNil(t, win.queueManager, "MessageDataがある選択実行時にqueueManagerが生成される")
		assert.True(t, win.queueManager.HasNext())
		assert.Same(t, followUp, win.queueManager.Dequeue())
	})
}

func Test_selectChoice(t *testing.T) {
	t.Parallel()

	t.Run("範囲外のインデックスは何もせずnilを返す", func(t *testing.T) {
		t.Parallel()

		win := &Window{isOpen: true, content: messageContent{Choices: []choiceOption{{Text: "A"}}}}

		err := win.selectChoice(-1)
		require.NoError(t, err)
		assert.True(t, win.IsOpen(), "範囲外の負のインデックスではウィンドウは閉じない")

		err = win.selectChoice(1)
		require.NoError(t, err)
		assert.True(t, win.IsOpen(), "範囲外の超過インデックスではウィンドウは閉じない")
	})

	t.Run("onChoiceコールバックに選択した選択肢を渡しウィンドウを閉じる", func(t *testing.T) {
		t.Parallel()

		var got choiceOption
		win := &Window{
			isOpen:  true,
			content: messageContent{Choices: []choiceOption{{Text: "A"}, {Text: "B"}}},
			onChoice: func(c choiceOption) {
				got = c
			},
		}

		err := win.selectChoice(1)

		require.NoError(t, err)
		assert.Equal(t, "B", got.Text)
		assert.True(t, win.IsClosed(), "選択後はウィンドウが閉じる")
	})

	t.Run("Actionがエラーを返すとウィンドウを閉じずにエラーを返す", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("action失敗")
		win := &Window{
			isOpen: true,
			content: messageContent{
				Choices: []choiceOption{{Text: "A", Action: func() error { return wantErr }}},
			},
		}

		err := win.selectChoice(0)

		require.ErrorIs(t, err, wantErr)
		assert.True(t, win.IsOpen(), "Actionがエラーの場合はウィンドウを閉じない")
	})
}

func Test_choiceBindings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(m *input.MockKeyboardInput)
		wantAction inputmapper.ActionID
		wantOK     bool
	}{
		{
			name:       "矢印上キーでメニュー上移動",
			setup:      func(m *input.MockKeyboardInput) { m.SetKeyPressedWithRepeat(ebiten.KeyArrowUp, true) },
			wantAction: inputmapper.ActionMenuUp,
			wantOK:     true,
		},
		{
			name:       "矢印下キーでメニュー下移動",
			setup:      func(m *input.MockKeyboardInput) { m.SetKeyPressedWithRepeat(ebiten.KeyArrowDown, true) },
			wantAction: inputmapper.ActionMenuDown,
			wantOK:     true,
		},
		{
			name:       "英字キーは移動しない",
			setup:      func(m *input.MockKeyboardInput) { m.SetKeyPressedWithRepeat(ebiten.KeyW, true) },
			wantAction: "",
			wantOK:     false,
		},
		{
			name:       "Enterの押下から押上のワンセットで選択",
			setup:      func(m *input.MockKeyboardInput) { m.SimulateEnterPressRelease() },
			wantAction: inputmapper.ActionMenuSelect,
			wantOK:     true,
		},
		{
			name:       "Escapeキーでキャンセル",
			setup:      func(m *input.MockKeyboardInput) { m.SetKeyJustPressed(ebiten.KeyEscape, true) },
			wantAction: inputmapper.ActionMenuCancel,
			wantOK:     true,
		},
		{
			name:       "何も入力が無ければ空を返す",
			setup:      func(_ *input.MockKeyboardInput) {},
			wantAction: "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := input.NewMockKeyboardInput()
			tt.setup(mock)

			action, ok := keybind.Convert(mock, choiceBindings)

			assert.Equal(t, tt.wantAction, action)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestWindow_HandleInput_未入力なら空を返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

	action, ok := win.HandleInput()

	assert.False(t, ok, "キーが押されていなければfalseを返す")
	assert.Equal(t, inputmapper.ActionID(""), action)
}
