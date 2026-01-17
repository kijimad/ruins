package messagewindow

import (
	"fmt"
	"image"
	"image/color"

	"github.com/ebitenui/ebitenui"
	eui_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	w "github.com/kijimaD/ruins/internal/world"
)

// Window はメッセージウィンドウを表す
type Window struct {
	config   Config
	content  MessageContent
	world    w.World
	onClose  func()
	onChoice func(choice Choice)

	isOpen      bool
	ui          *ebitenui.UI
	initialized bool
	window      *widget.Window

	// 選択肢がある場合、メニューシステムでページング可能な選択肢一覧を表示
	choiceMenuView  *tabmenu.View
	hasChoices      bool
	currentMenuPage int
	needsUIRebuild  bool // ページ変更時のUI再構築フラグ

	// 複数メッセージを順番に表示
	queueManager   *QueueManager
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
		w.initUI()
		w.initialized = true
	}

	if w.hasChoices && w.choiceMenuView != nil {
		if err := w.choiceMenuView.Update(); err != nil {
			return err
		}

		if w.needsUIRebuild {
			w.rebuildUI()
			w.needsUIRebuild = false
		}
	} else {
		if action, ok := w.HandleInput(); ok {
			w.DoAction(action)
		}
	}

	if w.ui != nil {
		w.ui.Update()
	}

	return nil
}

// Draw はウィンドウを描画する
func (w *Window) Draw(screen *ebiten.Image) {
	if !w.isOpen || w.ui == nil {
		return
	}

	w.ui.Draw(screen)
}

// IsOpen はウィンドウが開いているかを返す
func (w *Window) IsOpen() bool {
	return w.isOpen
}

// IsClosed はウィンドウが閉じているかを返す
func (w *Window) IsClosed() bool {
	return !w.isOpen
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

	// UI再初期化
	w.initialized = false
	w.ui = nil
	w.window = nil
	w.choiceMenuView = nil
}

// updateContentFromMessage はMessageDataから表示コンテンツを更新する
func (w *Window) updateContentFromMessage(msg *messagedata.MessageData) {
	w.content.SpeakerName = msg.Speaker
	w.content.TextSegmentLines = msg.TextSegmentLines

	w.content.Choices = make([]Choice, len(msg.Choices))
	for i, choice := range msg.Choices {
		w.content.Choices[i] = Choice{
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
						w.queueManager = NewQueueManager()
					}
					w.queueManager.EnqueueFront(choice.MessageData)
				}
				return nil
			},
		}
	}
}

// initUI はUIを初期化する
func (w *Window) initUI() {
	mainContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	w.ui = &ebitenui.UI{
		Container: mainContainer,
	}
	w.createAndAddWindow()
}

// createAndAddWindow はウィンドウを作成してUIに追加する
func (w *Window) createAndAddWindow() {
	// ウィンドウコンテナ
	windowContainer := w.createWindowContainer()

	// ウィンドウサイズを選択肢に応じて調整
	windowSize := w.calculateWindowSize()

	// タイトルバー付きウィンドウを作成
	titleContainer := w.createTitleContainer()
	w.window = styled.NewSmallWindow(
		titleContainer,
		windowContainer,
		widget.WindowOpts.CloseMode(widget.NONE), // クリックで閉じない
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.MinSize(windowSize.Width, windowSize.Height),
		widget.WindowOpts.MaxSize(windowSize.Width, windowSize.Height), // 固定サイズ
	)

	// ウィンドウ位置を計算
	x, y := w.calculateWindowPosition(windowSize)
	w.window.SetLocation(image.Rect(x, y, x+windowSize.Width, y+windowSize.Height))

	// UIにウィンドウを追加
	w.ui.AddWindow(w.window)
}

// createTitleContainer はタイトルコンテナを作成する
func (w *Window) createTitleContainer() *widget.Container {
	title := ""
	if w.content.SpeakerName != "" {
		title = w.content.SpeakerName
	}
	return styled.NewWindowHeaderContainer(title, w.world.Resources.UIResources)
}

// calculateWindowPosition はウィンドウの表示位置を計算する
func (w *Window) calculateWindowPosition(windowSize WindowSize) (x, y int) {
	screenWidth := w.world.Resources.ScreenDimensions.Width
	screenHeight := w.world.Resources.ScreenDimensions.Height

	x = (screenWidth - windowSize.Width) / 2

	numChoices := len(w.content.Choices)
	if w.hasChoices && numChoices > 0 {
		// メッセージがない場合は、より高い位置に配置
		if !w.hasMessage() {
			// メッセージなし: 中央より少し上に配置
			y = (screenHeight - windowSize.Height) / 2
			if numChoices > 5 {
				y = screenHeight / 3
			}
		} else {
			// メッセージあり: 選択肢数に応じて配置
			if numChoices <= 3 {
				y = (screenHeight - windowSize.Height) / 2
			} else if numChoices <= 8 {
				y = screenHeight / 3
			} else {
				y = screenHeight / 5
			}
		}
	} else {
		y = (screenHeight - windowSize.Height) / 2
	}

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
func (w *Window) calculateWindowSize() WindowSize {
	baseHeight := w.config.Size.Height

	// 選択肢がある場合は高さを再計算
	if w.hasChoices && len(w.content.Choices) > 0 {
		numChoices := len(w.content.Choices)

		// 構成要素の高さ
		messageHeight := 0 // メッセージの高さ
		if w.hasMessage() {
			messageHeight = 150 // メッセージがある場合のみ固定高さ
		}
		choiceItemHeight := 40 // 選択肢1つあたりの高さ
		choiceHeight := numChoices * choiceItemHeight
		topPadding := 20     // 上部パディング
		bottomPadding := 15  // 下部パディング
		titleBarHeight := 40 // タイトルバーの高さ
		spacingHeight := 10  // スペーサー間隔（メッセージがある場合のみ）
		if !w.hasMessage() {
			spacingHeight = 0
		}

		// 合計高さを計算
		calculatedHeight := messageHeight + choiceHeight + topPadding + bottomPadding + titleBarHeight + spacingHeight

		// 最低高さと最大高さを設定
		minHeightWithChoices := 300
		maxHeightWithChoices := int(float64(w.world.Resources.ScreenDimensions.Height) * 0.8) // 画面高さの80%まで

		if calculatedHeight < minHeightWithChoices {
			baseHeight = minHeightWithChoices
		} else if calculatedHeight > maxHeightWithChoices {
			baseHeight = maxHeightWithChoices
		} else {
			baseHeight = calculatedHeight
		}
	}

	return WindowSize{
		Width:  w.config.Size.Width,
		Height: baseHeight,
	}
}

// hasMessage はメッセージがあるかどうかを判定する
func (w *Window) hasMessage() bool {
	if len(w.content.TextSegmentLines) == 0 {
		return false
	}

	// すべての行が空かどうかをチェック
	for _, lineSegments := range w.content.TextSegmentLines {
		for _, seg := range lineSegments {
			// 空白文字以外があればメッセージありと判定
			for _, r := range seg.Text {
				if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
					return true
				}
			}
		}
	}

	return false
}

// createWindowContainer はウィンドウコンテナを作成する
func (w *Window) createWindowContainer() *widget.Container {
	windowContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(w.world.Resources.UIResources.Panel.ImageTrans),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				MaxHeight: 500,
			}),
		),
	)

	// 選択肢の有無でパディングを調整
	bottomPadding := 60
	if w.hasChoices && w.choiceMenuView != nil {
		bottomPadding = 15 // 選択肢がある場合は下部パディングを少なく
	}

	contentArea := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Padding(&widget.Insets{
					Top:    20,
					Bottom: bottomPadding,
					Left:   10,
					Right:  10,
				}),
				widget.RowLayoutOpts.Spacing(10),
			),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
		),
	)

	if w.hasChoices && w.choiceMenuView != nil {
		// 選択肢がある場合
		if w.hasMessage() {
			// メッセージがある場合: メッセージコンテナを表示
			messageContainer := widget.NewContainer(
				widget.ContainerOpts.Layout(
					widget.NewRowLayout(
						widget.RowLayoutOpts.Direction(widget.DirectionVertical),
						widget.RowLayoutOpts.Spacing(0),
					),
				),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{
						Stretch:   false,
						MaxHeight: 150, // 固定高さ
					}),
					widget.WidgetOpts.MinSize(0, 150),
				),
			)
			messageWidget := w.createSegmentedTextLines()
			messageContainer.AddChild(messageWidget)
			contentArea.AddChild(messageContainer)

			// スペーサー: メッセージと選択肢の間の空間を埋める
			spacer := widget.NewContainer(
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{
						Stretch: true,
					}),
				),
			)
			contentArea.AddChild(spacer)
		}

		// 選択肢コンテナ（自然な高さ、選択肢数に応じて伸びる）
		choicesContainer := w.createChoicesItemsContainer()
		contentArea.AddChild(choicesContainer)
	} else {
		// 選択肢がない場合: メッセージのみ表示
		messageWidget := w.createSegmentedTextLines()
		contentArea.AddChild(messageWidget)
	}

	windowContainer.AddChild(contentArea)

	if !w.hasChoices || w.choiceMenuView == nil {
		enterPrompt := w.createEnterPrompt()
		windowContainer.AddChild(enterPrompt)
	}

	return windowContainer
}

// createChoicesItemsContainer は選択肢アイテムのコンテナを作成する
func (w *Window) createChoicesItemsContainer() *widget.Container {
	var itemsContainer *widget.Container

	if w.choiceMenuView != nil {
		itemsContainer = w.choiceMenuView.BuildUI()
	} else {
		// 選択肢アイテムのコンテナ（縦並び、各アイテムを中央寄せ）
		itemsContainer = widget.NewContainer(
			widget.ContainerOpts.Layout(
				widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					widget.RowLayoutOpts.Spacing(5),
					widget.RowLayoutOpts.Padding(&widget.Insets{Top: 10, Bottom: 0}),
				),
			),
		)

		for i, choice := range w.content.Choices {
			// 各選択肢を中央配置するコンテナでラップ
			choiceWrapper := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
				widget.ContainerOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{
						Stretch: true,
					}),
				),
			)

			choiceText := w.createChoiceText(choice.Text, i == 0)
			choiceText.GetWidget().LayoutData = widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				StretchHorizontal:  false, // 水平方向に引き伸ばさない
			}
			choiceWrapper.AddChild(choiceText)
			itemsContainer.AddChild(choiceWrapper)
		}
	}

	// RowLayout内での配置（中央寄せ）
	itemsContainer.GetWidget().LayoutData = widget.RowLayoutData{
		Stretch:  false,
		Position: widget.RowLayoutPositionCenter,
	}

	return itemsContainer
}

// createEnterPrompt はEnterプロンプトを作成する
func (w *Window) createEnterPrompt() *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(0),
			),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding:            &widget.Insets{Bottom: 15},
			}),
		),
	)

	prompt := styled.NewListItemText(
		"Enter",
		color.RGBA{255, 255, 255, 255},
		true,
		w.world.Resources.UIResources,
	)

	container.AddChild(prompt)
	return container
}

// HandleInput はキーボード入力をActionに変換する
func (w *Window) HandleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	for _, key := range w.config.SkippableKeys {
		if key == ebiten.KeyEnter {
			if keyboardInput.IsEnterJustPressedOnce() {
				return inputmapper.ActionConfirm, true
			}
		} else if keyboardInput.IsKeyJustPressed(key) {
			return inputmapper.ActionSkip, true
		}
	}

	return "", false
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

	items := make([]tabmenu.Item, len(w.content.Choices))
	for i, choice := range w.content.Choices {
		items[i] = tabmenu.Item{
			ID:       choice.Text,
			Label:    choice.Text,
			Disabled: false,
			UserData: i,
		}
	}

	itemsPerPage := w.calculateItemsPerPage(len(w.content.Choices))

	// タブ1つ
	tabs := []tabmenu.TabItem{
		{
			ID:    "choices",
			Label: "",
			Items: items,
		},
	}

	config := tabmenu.Config{
		Tabs:             tabs,
		InitialTabIndex:  0,
		InitialItemIndex: 0,
		WrapNavigation:   true,
		ItemsPerPage:     itemsPerPage,
	}

	callbacks := tabmenu.Callbacks{
		OnSelectItem: func(_ int, itemIndex int, _ tabmenu.TabItem, _ tabmenu.Item) error {
			return w.selectChoice(itemIndex)
		},
		OnCancel: func() {
			w.Close()
		},
		OnItemChange: func(_ int, _, _ int, _ tabmenu.Item) error {
			if w.choiceMenuView != nil {
				newPage := w.choiceMenuView.GetCurrentPage()
				if newPage != w.currentMenuPage {
					w.currentMenuPage = newPage
					w.needsUIRebuild = true
				}
			}
			return nil
		},
	}

	// View を作成（TabMenu + UIBuilder を統合）
	w.choiceMenuView = tabmenu.NewView(config, callbacks, w.world)
	w.currentMenuPage = 1
}

// rebuildUI はUIを再構築する
func (w *Window) rebuildUI() {
	w.ui = nil
	w.window = nil
	w.initUI()
}

// calculateItemsPerPage は1ページあたりのアイテム数を計算する
func (w *Window) calculateItemsPerPage(totalItems int) int {
	windowHeight := w.calculateWindowSize().Height

	// メッセージがある場合のみメッセージエリアの高さを確保
	textAreaHeight := 0
	if w.hasMessage() {
		textAreaHeight = 150
	}
	pageIndicatorHeight := 30
	availableHeight := windowHeight - textAreaHeight - pageIndicatorHeight

	itemHeight := 35
	maxItemsPerPage := availableHeight / itemHeight

	if maxItemsPerPage < 3 {
		maxItemsPerPage = 3
	} else if maxItemsPerPage > 15 {
		maxItemsPerPage = 15
	}

	if totalItems <= maxItemsPerPage {
		return totalItems
	}

	return maxItemsPerPage
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

// createSegmentedTextLines はTextSegmentLinesから色付きテキストウィジェットを作成する（改行対応）
func (w *Window) createSegmentedTextLines() *widget.Container {
	// メインコンテナは縦方向（行を縦に並べる）
	mainContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(0),
				widget.RowLayoutOpts.Padding(&widget.Insets{
					Top:    0,
					Bottom: 0,
					Left:   8,
					Right:  8,
				}),
			),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)

	res := w.world.Resources.UIResources

	// 各行を処理
	for _, lineSegments := range w.content.TextSegmentLines {
		// 空行かどうかを判定（全セグメントが空白文字のみ）
		isEmptyLine := true
		for _, seg := range lineSegments {
			trimmed := ""
			for _, r := range seg.Text {
				if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
					trimmed += string(r)
				}
			}
			if trimmed != "" {
				isEmptyLine = false
				break
			}
		}

		// 空行の場合は固定高さのスペーサーを追加
		if isEmptyLine {
			spacer := widget.NewText(
				widget.TextOpts.Text(" ", &res.Text.Face, w.config.TextStyle.Color),
				widget.TextOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
				),
			)
			mainContainer.AddChild(spacer)
			continue
		}

		// 行コンテナ（横方向）
		lineContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(
				widget.NewRowLayout(
					widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
					widget.RowLayoutOpts.Spacing(0),
				),
			),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
			),
		)

		// 行内の各セグメントを処理
		for _, segment := range lineSegments {
			segmentColor := w.config.TextStyle.Color
			if segment.Color != nil {
				segmentColor = *segment.Color
			}

			// セグメント用のコンテナオプション
			var segmentOpts []widget.ContainerOpt
			segmentOpts = append(segmentOpts,
				widget.ContainerOpts.Layout(
					widget.NewRowLayout(
						widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
						widget.RowLayoutOpts.Spacing(0),
					),
				),
			)

			// 背景色の設定
			if segment.BackgroundColor != nil {
				segmentOpts = append(segmentOpts,
					widget.ContainerOpts.BackgroundImage(eui_image.NewNineSliceColor(*segment.BackgroundColor)),
				)
			}

			segmentContainer := widget.NewContainer(segmentOpts...)

			// テキストウィジェット
			textWidget := widget.NewText(
				widget.TextOpts.Text(segment.Text, &res.Text.Face, segmentColor),
				widget.TextOpts.WidgetOpts(
					widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
				),
			)

			segmentContainer.AddChild(textWidget)
			lineContainer.AddChild(segmentContainer)
		}

		mainContainer.AddChild(lineContainer)
	}

	return mainContainer
}

// createChoiceText は選択肢用のテキストウィジェットを作成する（自然な幅で中央寄せ用）
func (w *Window) createChoiceText(choiceText string, isSelected bool) *widget.Container {
	res := w.world.Resources.UIResources

	// 背景色の設定
	var backgroundColor *eui_image.NineSlice
	if isSelected {
		// 選択中は背景色を付ける
		backgroundColor = eui_image.NewNineSliceColor(color.NRGBA{R: 75, G: 104, B: 122, A: 100})
	} else {
		// 非選択は透明
		backgroundColor = eui_image.NewNineSliceColor(color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	}

	// コンテナ（自然な幅を保持）
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(backgroundColor),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// テキストウィジェット
	textWidget := widget.NewText(
		widget.TextOpts.Text(choiceText, &res.Text.Face, w.config.TextStyle.Color),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding: &widget.Insets{
					Top:    4,
					Bottom: 4,
					Left:   16,
					Right:  16,
				},
			}),
		),
	)

	container.AddChild(textWidget)
	return container
}
