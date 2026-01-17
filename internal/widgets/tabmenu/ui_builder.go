package tabmenu

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
)

// UIBuilder はTabMenuのUI要素を構築する
type uiBuilder struct {
	world       w.World
	itemWidgets []widget.PreferredSizeLocateableWidget // 現在表示中のウィジェット
}

// newUIBuilder はUIビルダーを作成する
func newUIBuilder(world w.World) *uiBuilder {
	return &uiBuilder{
		world:       world,
		itemWidgets: make([]widget.PreferredSizeLocateableWidget, 0),
	}
}

// BuildUI はtabMenuのUI要素を構築する（タブが1つの場合を想定）
func (b *uiBuilder) BuildUI(tabMenu *tabMenu) *widget.Container {
	// タブが1つしかない場合は、そのタブのアイテムを直接表示
	// 垂直リスト表示（固定）
	return b.buildVerticalUI(tabMenu)
}

// buildVerticalUI は垂直リスト表示のUIを構築する
func (b *uiBuilder) buildVerticalUI(tabMenu *tabMenu) *widget.Container {
	mainContainer := styled.NewVerticalContainer()

	// ページインジケーターを追加
	pageText := tabMenu.GetPageIndicatorText()
	if pageText != "" {
		pageIndicator := b.CreatePageIndicator(tabMenu)
		mainContainer.AddChild(pageIndicator)
	}

	// メニューアイテムのコンテナ
	menuContainer := styled.NewVerticalContainer()
	b.itemWidgets = make([]widget.PreferredSizeLocateableWidget, 0)

	// 表示する項目のみを追加（スクロール対応）
	visibleItems, indices := tabMenu.GetVisibleItems()
	for i, item := range visibleItems {
		originalIndex := indices[i]
		btn := b.CreateMenuButton(tabMenu, originalIndex, item)
		menuContainer.AddChild(btn)
		b.itemWidgets = append(b.itemWidgets, btn)
	}

	mainContainer.AddChild(menuContainer)
	b.UpdateFocus(tabMenu)

	return mainContainer
}

// CreateMenuButton はメニューボタンを作成する
// 追加ラベルがある場合は、Button + ラベル群をコンテナでまとめて返す
func (b *uiBuilder) CreateMenuButton(tabMenu *tabMenu, index int, item Item) widget.PreferredSizeLocateableWidget {
	res := b.world.Resources.UIResources

	// フォーカス状態をチェック
	isFocused := index == tabMenu.GetCurrentItemIndex()

	// ボタン画像を作成
	var buttonImage *widget.ButtonImage
	if isFocused {
		buttonImage = b.createFocusedButtonImage()
	} else {
		buttonImage = b.createTransparentButtonImage()
	}

	// ボタンを作成
	btn := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: len(item.AdditionalLabels) == 0, // 追加ラベルがなければStretch
			}),
			widget.WidgetOpts.MinSize(100, 28),
		),
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.Text(item.Label, &res.Button.Face, res.Button.Text),
		widget.ButtonOpts.TextPadding(&res.Button.Padding),
		widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		widget.ButtonOpts.ClickedHandler(func(_ *widget.ButtonClickedEventArgs) {
			if err := tabMenu.SetItemIndex(index); err != nil {
				panic(err)
			}
			if err := tabMenu.selectCurrentItem(); err != nil {
				panic(err)
			}
		}),
	)

	// 無効化されたアイテムの処理
	if item.Disabled {
		btn.GetWidget().Disabled = true
	}

	// 追加ラベルがない場合はボタンをそのまま返す
	if len(item.AdditionalLabels) == 0 {
		return btn
	}

	// 追加ラベルがある場合は、コンテナでまとめる
	container := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
		)),
	)

	// ボタンを追加
	container.AddChild(btn)

	// 右側: 追加ラベル群（右寄せ）
	for _, label := range item.AdditionalLabels {
		additionalText := widget.NewText(
			widget.TextOpts.Text(label, &res.Button.Face, res.Button.Text.Idle),
			widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Position: widget.RowLayoutPositionEnd,
				}),
			),
		)
		container.AddChild(additionalText)
	}

	return container
}

// UpdateFocus はメニューのフォーカス表示を更新する
// カーソルで選択中の要素だけボタンを変える。マウスのhoverでは色が変わらないようにしている
// カーソル移動は独自実装なので、UIを対応させるために必要
func (b *uiBuilder) UpdateFocus(tabMenu *tabMenu) {
	if len(b.itemWidgets) == 0 {
		return
	}

	// 表示中の項目とそのインデックスを取得
	_, indices := tabMenu.GetVisibleItems()

	// 全てのボタンのフォーカスを更新
	for i, w := range b.itemWidgets {
		if i >= len(indices) {
			continue
		}

		originalIndex := indices[i]
		isFocused := originalIndex == tabMenu.GetCurrentItemIndex()

		var btn *widget.Button

		// ボタンの場合
		if b, ok := w.(*widget.Button); ok {
			btn = b
		} else if container, ok := w.(*widget.Container); ok {
			// コンテナの場合は、子要素からボタンを探す
			for _, child := range container.Children() {
				if b, ok := child.(*widget.Button); ok {
					btn = b
					break
				}
			}
		}

		// ボタンが見つかった場合、フォーカス状態に応じて画像を更新
		if btn != nil {
			if isFocused {
				// フォーカス時: より明るい背景色
				focusedImage := b.createFocusedButtonImage()
				btn.SetImage(focusedImage)
			} else {
				// 非フォーカス時: 通常の半透明背景
				normalImage := b.createTransparentButtonImage()
				btn.SetImage(normalImage)
			}
		}
	}
}

// createTransparentButtonImage は半透明のボタン画像を作成する
// マウスホバーでは色が変わらず、キーボード操作（フォーカス）でのみ反応する
func (b *uiBuilder) createTransparentButtonImage() *widget.ButtonImage {
	// アイドル状態: 透明
	idle := image.NewNineSliceColor(color.NRGBA{R: 0, G: 0, B: 0, A: 0})

	// プレス状態: さらに明るい半透明の灰色
	pressed := image.NewNineSliceColor(color.NRGBA{R: 100, G: 100, B: 100, A: 180})

	// 無効状態: 暗い半透明
	disabled := image.NewNineSliceColor(color.NRGBA{R: 30, G: 30, B: 30, A: 16})

	return &widget.ButtonImage{
		Idle:     idle,
		Hover:    idle, // フォーカス時と同じ色でマウスホバー効果を無効化
		Pressed:  pressed,
		Disabled: disabled,
	}
}

// createFocusedButtonImage はフォーカス時の明るいボタン画像を作成する
func (b *uiBuilder) createFocusedButtonImage() *widget.ButtonImage {
	// フォーカス時: 半透明の灰色
	focused := image.NewNineSliceColor(consts.ButtonHoverColor)

	// プレス状態: さらに明るい色
	pressed := image.NewNineSliceColor(consts.ButtonPressedColor)

	// 無効状態: 暗い半透明
	disabled := image.NewNineSliceColor(color.NRGBA{R: 30, G: 30, B: 30, A: 80})

	return &widget.ButtonImage{
		Idle:     focused,
		Hover:    focused, // フォーカス時と同じ色でマウスホバー効果を無効化
		Pressed:  pressed,
		Disabled: disabled,
	}
}

// CreatePageIndicator はページインジケーターを作成する
func (b *uiBuilder) CreatePageIndicator(tabMenu *tabMenu) *widget.Text {
	res := b.world.Resources.UIResources

	pageText := tabMenu.GetPageIndicatorText()

	return widget.NewText(
		widget.TextOpts.Text(pageText, &res.Text.SmallFace, color.White),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 20),
		),
	)
}

// UpdateTabDisplayContainer はタブ表示コンテナを更新する
// ページインジケーター、アイテム一覧、空の場合のメッセージを表示する
func (b *uiBuilder) UpdateTabDisplayContainer(container *widget.Container, tabMenu *tabMenu) {
	// 既存の子要素をクリア
	container.RemoveChildren()

	currentTab := tabMenu.GetCurrentTab()
	currentItemIndex := tabMenu.GetCurrentItemIndex()

	// ページインジケーターを表示
	pageText := tabMenu.GetPageIndicatorText()
	if pageText != "" {
		pageIndicator := styled.NewPageIndicator(pageText, b.world.Resources.UIResources)
		container.AddChild(pageIndicator)
	}

	// 現在のページで表示されるアイテムとインデックスを取得
	visibleItems, indices := tabMenu.GetVisibleItems()

	// アイテム一覧を表示（ページ内のアイテムのみ）
	for i, item := range visibleItems {
		actualIndex := indices[i]
		isSelected := actualIndex == currentItemIndex && currentItemIndex >= 0

		// Disabledアイテムの場合は灰色で表示
		if item.Disabled {
			itemWidget := styled.NewListItemText(item.Label, consts.ForegroundColor, isSelected, b.world.Resources.UIResources, item.AdditionalLabels...)
			container.AddChild(itemWidget)
		} else if isSelected {
			// 選択中のアイテムは背景色付きで明るい文字色
			itemWidget := styled.NewListItemText(item.Label, consts.TextColor, true, b.world.Resources.UIResources, item.AdditionalLabels...)
			container.AddChild(itemWidget)
		} else {
			// 非選択のアイテムは背景なしで明るい文字色
			itemWidget := styled.NewListItemText(item.Label, consts.TextColor, false, b.world.Resources.UIResources, item.AdditionalLabels...)
			container.AddChild(itemWidget)
		}
	}

	// アイテムがない場合の表示
	if len(currentTab.Items) == 0 {
		emptyText := styled.NewDescriptionText("(アイテムなし)", b.world.Resources.UIResources)
		container.AddChild(emptyText)
	}
}
