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
	"github.com/kijimaD/ruins/internal/widgets/uicore"
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

	t.Run("内容が最小高に収まるなら最小高のままにする", func(t *testing.T) {
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

		// 余白36 + 選択肢26*1 = 62 で最小高300に収まるので、窓は痩せず最小高のまま
		assert.Equal(t, MinHeight, size.Height)
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

		// 本文150 + タイトル25 + 余白52 + 選択肢26*2 = 279 で最小高300に収まる
		assert.Equal(t, MinHeight, size.Height)
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

		// 画面高720*0.8=576 から取り分を引いた残りを選択肢の行高26で割る。
		// 取り分は本文150 + タイトル25 + 余白52 + ページ表示24 = 251 で、(576-251)/26 は 12
		assert.Equal(t, 12, got)
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

// fixedInputSource は常に同じ Action を返す入力供給源を作る
func fixedInputSource(action inputmapper.ActionID, ok bool) inputmapper.Source {
	return func() (inputmapper.ActionID, bool) { return action, ok }
}

// sequenceInputSource は呼び出しごとに actions を1つずつ消費し、尽きたら入力無しを返す
func sequenceInputSource(actions ...inputmapper.ActionID) inputmapper.Source {
	i := 0
	return func() (inputmapper.ActionID, bool) {
		if i >= len(actions) {
			return "", false
		}
		a := actions[i]
		i++
		return a, true
	}
}

func Test_initChoiceMenu(t *testing.T) {
	t.Parallel()

	t.Run("選択肢がある場合はタブ項目とストアを構築する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		msg := messagedata.NewDialogMessage("どうする？", "NPC").
			WithChoice("はい", func(_ w.World) error { return nil }).
			WithChoice("いいえ", nil)
		win := NewWindow(world, msg)

		win.initChoiceMenu()

		require.Len(t, win.choiceConfig.Tabs, 1)
		items := win.choiceConfig.Tabs[0].Items
		require.Len(t, items, 2)
		assert.Equal(t, "はい", items[0].Label)
		assert.Equal(t, "いいえ", items[1].Label)
		assert.Equal(t, 0, items[0].UserData)
		assert.Equal(t, 1, items[1].UserData)
		require.NotNil(t, win.choiceStore, "選択肢のナビゲーション状態を持つストアを作る")
		assert.Equal(t, 0, win.choiceState.ItemIndex, "初期カーソルは先頭")
	})

	t.Run("選択肢が無い場合は何もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		win.initChoiceMenu()

		assert.Nil(t, win.choiceStore, "選択肢が無いのでストアを作らない")
		assert.Nil(t, win.choiceConfig.Tabs, "選択肢が無いのでタブ項目を組まない")
	})
}

func TestWindow_Update(t *testing.T) {
	t.Parallel()

	t.Run("閉じている場合は初期化も描画ツリー構築もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))
		win.isOpen = false

		err := win.Update()

		require.NoError(t, err)
		assert.Nil(t, win.body)
		assert.False(t, win.initialized)
	})

	t.Run("選択肢が無く入力が無ければ開いたままEnterプロンプトを組む", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		world.Resources.InputSource = fixedInputSource("", false)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		err := win.Update()

		require.NoError(t, err)
		assert.True(t, win.IsOpen())
		require.NotNil(t, win.body)
		assert.Contains(t, uicore.CollectLabels(win.body), "Enter")
	})

	t.Run("選択肢が無くConfirmが来ると閉じる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		world.Resources.InputSource = fixedInputSource(inputmapper.ActionConfirm, true)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		err := win.Update()

		require.NoError(t, err)
		assert.True(t, win.IsClosed())
	})

	t.Run("選択肢があり入力が無ければ状態を変えない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		world.Resources.InputSource = fixedInputSource("", false)
		msg := messagedata.NewDialogMessage("どうする？", "NPC").WithChoice("はい", nil)
		win := NewWindow(world, msg)

		err := win.Update()

		require.NoError(t, err)
		assert.True(t, win.IsOpen())
		assert.Equal(t, 0, win.choiceState.ItemIndex)
	})

	t.Run("選択肢がありカーソル移動後に選択すると対象のActionを実行して閉じる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		var called0, called1 bool
		msg := messagedata.NewDialogMessage("どうする？", "NPC").
			WithChoice("はい", func(_ w.World) error { called0 = true; return nil }).
			WithChoice("いいえ", func(_ w.World) error { called1 = true; return nil })
		win := NewWindow(world, msg)
		world.Resources.InputSource = sequenceInputSource(inputmapper.ActionMenuDown, inputmapper.ActionMenuSelect)

		require.NoError(t, win.Update())
		assert.Equal(t, 1, win.choiceState.ItemIndex, "下移動でカーソルが2件目に進む")
		assert.True(t, win.IsOpen())

		require.NoError(t, win.Update())

		assert.True(t, win.IsClosed())
		assert.False(t, called0, "1件目のActionは実行されない")
		assert.True(t, called1, "選択した2件目のActionが実行される")
	})

	t.Run("選択肢がありEscapeが来ると選択を実行せず閉じる", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		var called bool
		msg := messagedata.NewDialogMessage("どうする？", "NPC").
			WithChoice("はい", func(_ w.World) error { called = true; return nil })
		win := NewWindow(world, msg)
		world.Resources.InputSource = fixedInputSource(inputmapper.ActionMenuCancel, true)

		require.NoError(t, win.Update())

		assert.True(t, win.IsClosed())
		assert.False(t, called, "キャンセルでは選択肢のActionを実行しない")
	})
}

func TestWindow_Draw(t *testing.T) {
	t.Parallel()

	t.Run("閉じている場合は描画しない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))
		require.NoError(t, win.Update())
		win.isOpen = false

		sd := world.Resources.ScreenDimensions
		screen := ebiten.NewImage(sd.Width, sd.Height)
		win.Draw(screen)

		_, _, _, a := screen.At(sd.Width/2, sd.Height/2).RGBA()
		assert.Equal(t, uint32(0), a, "閉じているので何も描かれない")
	})

	t.Run("bodyが未構築の場合は描画しない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))

		sd := world.Resources.ScreenDimensions
		screen := ebiten.NewImage(sd.Width, sd.Height)
		win.Draw(screen)

		_, _, _, a := screen.At(sd.Width/2, sd.Height/2).RGBA()
		assert.Equal(t, uint32(0), a, "bodyが無いので何も描かれない")
	})

	t.Run("開いていてbodyがあれば窓の背景を描く", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t, testutil.WithUI())
		world.Resources.InputSource = fixedInputSource("", false)
		win := NewWindow(world, messagedata.NewSystemMessage("テスト"))
		require.NoError(t, win.Update())

		sd := world.Resources.ScreenDimensions
		screen := ebiten.NewImage(sd.Width, sd.Height)
		win.Draw(screen)

		size := win.calculateWindowSize()
		x, y := win.calculateWindowPosition(size)
		_, _, _, a := screen.At(x+size.Width/2, y+size.Height/2).RGBA()
		assert.NotEqual(t, uint32(0), a, "窓の中心には背景が描かれている")
	})
}
