package messagewindow

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// Window はメッセージウィンドウを表す。
//
// 描画は uicore のツリーで組み、グローバル可変状態に触れない。UI は保持せず
// 毎フレーム buildTree で組み直す。フォーカスやページの差分追跡は不要で、現在の状態から都度描く
type Window struct {
	config      windowConfig
	content     messageContent
	world       w.World
	onClose     func()
	onChoice    func(choice choiceOption)
	isOpen      bool
	initialized bool
	body        uicore.Widget

	// 選択肢がある場合、ページング可能な選択肢一覧を表示する。ナビゲーションは hooks Store が持ち、
	// 描画に要る現在位置を choiceState に写す
	hasChoices   bool
	choiceConfig tabMenuConfig
	choiceState  viewState
	choiceStore  *hooks.Store

	// 複数メッセージを順番に表示
	queueManager   *queueManager
	currentMessage *messagedata.MessageData
}

// Update はウィンドウを更新する
func (w *Window) Update() error {
	if !w.isOpen {
		return nil
	}

	if !w.initialized {
		w.hasChoices = len(w.content.Choices) > 0
		if w.hasChoices {
			w.initChoiceMenu()
		}
		w.initialized = true
	}

	if w.hasChoices {
		if err := w.handleChoiceInput(); err != nil {
			return err
		}
	} else {
		if action, ok := w.HandleInput(); ok {
			w.DoAction(action)
		}
	}

	// 状態が確定してから現在の見た目を組み直す。閉じた、または次メッセージへ切り替えた直後は
	// 組まずに次フレームの再初期化へ譲る
	if w.isOpen && w.initialized {
		w.body = w.buildTree()
	}

	return nil
}

// Draw はウィンドウを描画する
func (w *Window) Draw(screen *ebiten.Image) {
	if !w.isOpen || w.body == nil {
		return
	}
	w.body.Draw(uicore.NewEbitenCanvas(screen))
}

// IsOpen はウィンドウが開いているかを返す
func (w *Window) IsOpen() bool {
	return w.isOpen
}

// IsClosed はウィンドウが閉じているかを返す
func (w *Window) IsClosed() bool {
	return !w.isOpen
}

// CurrentMessage は現在表示中のメッセージデータを返す
func (w *Window) CurrentMessage() *messagedata.MessageData {
	return w.currentMessage
}

// Close はウィンドウを閉じる
// キューに次のメッセージがある場合は閉じずに次を表示する
func (w *Window) Close() {
	if !w.isOpen {
		return
	}

	if w.currentMessage != nil && w.currentMessage.OnComplete != nil {
		w.currentMessage.OnComplete()
	}

	if w.queueManager != nil && w.queueManager.HasNext() {
		w.showNextMessage()
		return
	}

	w.isOpen = false

	if w.onClose != nil {
		w.onClose()
	}
}

// showNextMessage はキューから次のメッセージを取り出して表示する
func (w *Window) showNextMessage() {
	if w.queueManager == nil {
		return
	}

	nextMessage := w.queueManager.Dequeue()
	if nextMessage == nil {
		w.isOpen = false
		if w.onClose != nil {
			w.onClose()
		}
		return
	}

	w.currentMessage = nextMessage
	w.updateContentFromMessage(nextMessage)

	// 次フレームで再初期化させる
	w.initialized = false
	w.body = nil
	w.choiceStore = nil
}

// updateContentFromMessage はMessageDataから表示コンテンツを更新する
func (w *Window) updateContentFromMessage(msg *messagedata.MessageData) {
	w.content.SpeakerName = msg.Speaker
	w.content.TextSegmentLines = msg.TextSegmentLines

	w.content.Choices = make([]choiceOption, len(msg.Choices))
	for i, choice := range msg.Choices {
		w.content.Choices[i] = choiceOption{
			Text: choice.Text,
			Action: func() error {
				if choice.Action != nil {
					if err := choice.Action(w.world); err != nil {
						return err
					}
				}
				// 選択肢に関連メッセージがある場合はキュー先頭に追加して即座に表示
				if choice.MessageData != nil {
					if w.queueManager == nil {
						w.queueManager = newQueueManager()
					}
					w.queueManager.EnqueueFront(choice.MessageData)
				}
				return nil
			},
		}
	}
}

// calculateWindowPosition はウィンドウの表示位置を計算する。
// 横は画面中央、上端は他のウィンドウと同じ MenuWindowTop で揃える
func (w *Window) calculateWindowPosition(windowSize windowSize) (x, y int) {
	screenWidth := w.world.Resources.ScreenDimensions.Width
	screenHeight := w.world.Resources.ScreenDimensions.Height

	x = (screenWidth - windowSize.Width) / 2
	y = theme.MenuWindowTop

	margin := 30
	if y+windowSize.Height > screenHeight-margin {
		y = screenHeight - windowSize.Height - margin
	}
	if y < margin {
		y = margin
	}

	return x, y
}

// calculateWindowSize は選択肢に応じてウィンドウサイズを計算する
func (w *Window) calculateWindowSize() windowSize {
	// 内容が収まる高さと設定の最小高の大きいほうを採る。内容が少なくても窓は痩せず、
	// 選択肢が増えれば必要なぶんだけ伸びる
	height := max(w.config.Size.Height, w.requiredHeight())
	// 画面高の8割で頭を打つ。窓が画面を覆い尽くさないようにする
	height = min(height, int(float64(w.world.Resources.ScreenDimensions.Height)*maxHeightRatio))

	return windowSize{
		Width:  w.config.Size.Width,
		Height: height,
	}
}

// hasMessage はメッセージがあるかどうかを判定する
func (w *Window) hasMessage() bool {
	for _, lineSegments := range w.content.TextSegmentLines {
		if !lineIsBlank(lineSegments) {
			return true
		}
	}
	return false
}

// HandleInput はキーボード入力をActionに変換する。束縛表は設定の SkippableKeys から導出し、
// 供給源があればそこから読むので再生ドライバでも駆動できる
func (w *Window) HandleInput() (inputmapper.ActionID, bool) {
	return keybind.ReadInput(w.world, skipBindings(w.config.SkippableKeys))
}

// skipBindings は読み飛ばしキーの一覧から束縛表を導出する。Enter だけは確定として区別する
func skipBindings(keys []ebiten.Key) []keybind.Binding {
	rows := make([]keybind.Binding, 0, len(keys))
	for _, key := range keys {
		action := inputmapper.ActionSkip
		if key == ebiten.KeyEnter {
			action = inputmapper.ActionConfirm
		}
		rows = append(rows, keybind.Binding{Key: key, Action: action})
	}
	return rows
}

// DoAction はActionを実行する
func (w *Window) DoAction(action inputmapper.ActionID) {
	switch action {
	case inputmapper.ActionConfirm, inputmapper.ActionSkip:
		w.Close()
	default:
		panic(fmt.Sprintf("invalid action: %s", action))
	}
}

// initChoiceMenu は選択肢メニューを初期化する
func (w *Window) initChoiceMenu() {
	if len(w.content.Choices) == 0 {
		return
	}

	items := make([]item, len(w.content.Choices))
	for i, choice := range w.content.Choices {
		items[i] = item{
			ID:       choice.Text,
			Label:    choice.Text,
			UserData: i,
		}
	}

	itemsPerPage := w.calculateItemsPerPage(len(w.content.Choices))

	w.choiceConfig = tabMenuConfig{
		Tabs: []tabItem{
			{ID: "choices", Label: "", Items: items},
		},
		ItemsPerPage: itemsPerPage,
	}

	// hooks で Skip 込みのナビゲーションを管理する
	skips := make([]bool, len(w.currentMessage.Choices))
	for i, choice := range w.currentMessage.Choices {
		skips[i] = choice.Action == nil
	}

	w.choiceStore = hooks.NewStore()
	menuState := hooks.UseTabMenu(w.choiceStore, "choices", hooks.TabMenuConfig{
		TabCount:     1,
		ItemCounts:   []int{len(items)},
		ItemsPerPage: itemsPerPage,
		Skips:        [][]bool{skips},
	})

	w.choiceState = viewState{
		TabIndex:  menuState.TabIndex,
		ItemIndex: menuState.ItemIndex,
	}
}

// handleChoiceInput はキー入力を hooks Store にディスパッチし、描画状態を同期する
func (w *Window) handleChoiceInput() error {
	action, ok := keybind.ReadInput(w.world, choiceBindings)
	if !ok {
		return nil
	}

	switch action {
	case inputmapper.ActionMenuSelect:
		menuState, _ := hooks.GetStoreState[hooks.TabMenuState](w.choiceStore, "choices")
		return w.selectChoice(menuState.ItemIndex)
	case inputmapper.ActionMenuCancel:
		w.Close()
		return nil
	default:
		w.choiceStore.Dispatch(action)
		menuState, _ := hooks.GetStoreState[hooks.TabMenuState](w.choiceStore, "choices")
		w.choiceState = viewState{
			TabIndex:  menuState.TabIndex,
			ItemIndex: menuState.ItemIndex,
		}
		return nil
	}
}

// choiceBindings は選択肢メニューの束縛表
var choiceBindings = []keybind.Binding{
	{Key: ebiten.KeyArrowUp, Press: keybind.PressRepeat, Action: inputmapper.ActionMenuUp},
	{Key: ebiten.KeyArrowDown, Press: keybind.PressRepeat, Action: inputmapper.ActionMenuDown},
	{Key: ebiten.KeyEnter, Action: inputmapper.ActionMenuSelect},
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionMenuCancel},
}

// calculateItemsPerPage は1ページあたりの選択肢の件数を返す。窓の上限高から選択肢以外の
// 取り分を引いた残りを、実際に描く行高で割る。取り分と行高は描画と同じ値を使うので、
// 計算した件数がそのまま収まる。
func (w *Window) calculateItemsPerPage(totalItems int) int {
	maxHeight := int(float64(w.world.Resources.ScreenDimensions.Height) * maxHeightRatio)

	// 選択肢の塊を除いた取り分。requiredHeight から選択肢1行ぶんを引いて求める
	overhead := w.chromeHeight() + pageRowH
	perPage := (maxHeight - overhead) / choiceRowH
	perPage = min(max(perPage, minItemsPerPage), maxItemsPerPage)

	return min(totalItems, perPage)
}

// selectChoice は選択肢を選択する
func (w *Window) selectChoice(index int) error {
	if index < 0 || index >= len(w.content.Choices) {
		return nil
	}

	choice := w.content.Choices[index]

	// コールバック実行
	if w.onChoice != nil {
		w.onChoice(choice)
	}

	// アクション実行
	if choice.Action != nil {
		if err := choice.Action(); err != nil {
			return err
		}
	}

	// ウィンドウを閉じる
	w.Close()
	return nil
}
