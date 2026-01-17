package styled

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
)

// NewRowContainer は汎用的なrowコンテナを作成する
func NewRowContainer(opts ...widget.ContainerOpt) *widget.Container {
	return widget.NewContainer(
		append([]widget.ContainerOpt{
			widget.ContainerOpts.Layout(
				widget.NewRowLayout(
					BaseRowLayoutOpts()...,
				),
			),
		}, opts...)...,
	)
}

// NewVerticalContainer は中身が縦並びのコンテナを作成する
func NewVerticalContainer(opts ...widget.ContainerOpt) *widget.Container {
	return widget.NewContainer(
		append([]widget.ContainerOpt{
			widget.ContainerOpts.Layout(
				widget.NewRowLayout(
					append([]widget.RowLayoutOpt{
						widget.RowLayoutOpts.Direction(widget.DirectionVertical),
					},
						BaseRowLayoutOpts()...,
					)...,
				),
			),
		}, opts...)...,
	)
}

// NewItemGridContainer はアイテム系メニューのRootとなる3x3のグリッドコンテナを作成する
func NewItemGridContainer(opts ...widget.ContainerOpt) *widget.Container {
	return widget.NewContainer(
		append([]widget.ContainerOpt{
			widget.ContainerOpts.Layout(
				widget.NewGridLayout(
					// アイテム, スクロール, アイテム性能で3列になっている
					widget.GridLayoutOpts.Columns(3),
					widget.GridLayoutOpts.Spacing(4, 4),
					widget.GridLayoutOpts.Stretch([]bool{true, false, true}, []bool{false, true, false}),
					widget.GridLayoutOpts.Padding(&widget.Insets{
						Top:    4,
						Bottom: 4,
						Left:   4,
						Right:  4,
					}),
				)),
		}, opts...)...,
	)
}

// NewVSplitContainer は縦分割コンテナを作成する
func NewVSplitContainer(top *widget.Container, bottom *widget.Container, opts ...widget.ContainerOpt) *widget.Container {
	split := widget.NewContainer(
		append([]widget.ContainerOpt{
			widget.ContainerOpts.Layout(
				widget.NewGridLayout(
					widget.GridLayoutOpts.Columns(1),
					widget.GridLayoutOpts.Spacing(4, 4),
					widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true, true}),
					widget.GridLayoutOpts.Padding(&widget.Insets{
						Top:    4,
						Bottom: 4,
						Left:   4,
						Right:  4,
					}),
				)),
		}, opts...)...,
	)
	split.AddChild(top)
	split.AddChild(bottom)

	return split
}

// NewWSplitContainer は横分割コンテナを作成する
func NewWSplitContainer(right *widget.Container, left *widget.Container, opts ...widget.ContainerOpt) *widget.Container {
	split := widget.NewContainer(
		append([]widget.ContainerOpt{
			widget.ContainerOpts.Layout(
				widget.NewGridLayout(
					widget.GridLayoutOpts.Columns(2),
					widget.GridLayoutOpts.Spacing(4, 4),
					widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true}),
					widget.GridLayoutOpts.Padding(&widget.Insets{
						Top:    4,
						Bottom: 4,
						Left:   4,
						Right:  4,
					}),
				)),
		}, opts...)...,
	)
	split.AddChild(right)
	split.AddChild(left)

	return split
}

// NewWindowContainer はウィンドウの本体を作成する
func NewWindowContainer(res *resources.UIResources) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{
				Top:    20,
				Bottom: 20,
				Left:   10,
				Right:  10,
			}),
			widget.RowLayoutOpts.Spacing(2),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				MaxHeight: 500,
			}),
		),
	)
}

// NewWindowHeaderContainer はウィンドウのヘッダーを作成する
func NewWindowHeaderContainer(title string, res *resources.UIResources) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.TitleBar),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	container.AddChild(widget.NewText(
		widget.TextOpts.Text(title, &res.Text.Face, consts.TextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			Padding: &widget.Insets{
				Top:    4,
				Bottom: 4,
				Left:   8,
				Right:  8,
			},
		})),
	))

	return container
}

// text ================

// NewMenuText は汎用メニューテキストを作成する
func NewMenuText(title string, res *resources.UIResources) *widget.Text {
	text := widget.NewText(
		widget.TextOpts.Text(title, &res.Text.Face, consts.TextColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	)

	return text
}

// NewTitleText はタイトル用テキストを作成する（大きめ、目立つ）
func NewTitleText(text string, res *resources.UIResources) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(text, &res.Text.TitleFace, consts.TextColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	)
}

// NewDescriptionText は説明文用テキストを作成する（小さめ、補助的）
func NewDescriptionText(text string, res *resources.UIResources) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(text, &res.Text.SmallFace, consts.ForegroundColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	)
}

// NewPageIndicator は右寄せのページインジケーターを作成する
func NewPageIndicator(text string, res *resources.UIResources) *widget.Container {
	// 透明な背景のコンテナを作成（NewListItemTextと同じパターン）
	backgroundColor := image.NewNineSliceColor(consts.TransparentColor)

	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(backgroundColor),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true, // 横幅を親コンテナに合わせる
			}),
			widget.WidgetOpts.MinSize(120, 0), // NewListItemTextと同じ最小横幅
		),
	)

	// 右寄せのテキスト
	textWidget := widget.NewText(
		widget.TextOpts.Text(text, &res.Text.SmallFace, consts.ForegroundColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd, // 右寄せ
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding: &widget.Insets{
					Top:    2,
					Bottom: 2,
					Left:   8,
					Right:  8,
				},
			}),
		),
	)

	container.AddChild(textWidget)
	return container
}

// NewBodyText は本文用テキストを作成する
func NewBodyText(title string, _ color.RGBA, res *resources.UIResources) *widget.Text {
	text := widget.NewText(
		widget.TextOpts.Text(title, &res.Text.Face, consts.TextColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	)

	return text
}

// NewListItemText はリスト項目用テキストを作成する（背景色変更で選択状態を表現）
// additionalLabels が空の場合は単純なテキスト表示、指定された場合は右側に追加ラベルを表示
func NewListItemText(text string, textColor color.RGBA, isSelected bool, res *resources.UIResources, additionalLabels ...string) *widget.Container {
	// 背景色の設定
	var backgroundColor *image.NineSlice
	if isSelected {
		// 選択中は背景色を付ける
		backgroundColor = image.NewNineSliceColor(consts.ButtonHoverColor)
	} else {
		// 非選択は背景なし（透明）
		backgroundColor = image.NewNineSliceColor(consts.TransparentColor)
	}

	// 追加ラベルがない場合は、外側のcontainerに背景色を設定
	var containerOpts []widget.ContainerOpt
	if len(additionalLabels) == 0 {
		containerOpts = []widget.ContainerOpt{
			widget.ContainerOpts.BackgroundImage(backgroundColor),
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(10),
				widget.RowLayoutOpts.Padding(&widget.Insets{
					Top:    0,
					Bottom: 0,
					Left:   8,
					Right:  8,
				}),
			)),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
				widget.WidgetOpts.MinSize(120, 0),
			),
		}
	} else {
		containerOpts = []widget.ContainerOpt{
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(10),
				widget.RowLayoutOpts.Padding(&widget.Insets{
					Top:    0,
					Bottom: 0,
					Left:   8,
					Right:  8,
				}),
			)),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
				widget.WidgetOpts.MinSize(120, 0),
			),
		}
	}

	container := widget.NewContainer(containerOpts...)

	// メインテキストコンテナ
	// 追加ラベルがない場合はStretch、ある場合は固定幅
	var mainTextContainerOpts []widget.ContainerOpt
	if len(additionalLabels) == 0 {
		// 追加ラベルなし: 全幅使用、背景色なし（外側のcontainerに設定済み）
		mainTextContainerOpts = []widget.ContainerOpt{
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Stretch: true,
				}),
			),
		}
	} else {
		// 追加ラベルあり: 固定幅、背景色あり
		mainTextContainerOpts = []widget.ContainerOpt{
			widget.ContainerOpts.BackgroundImage(backgroundColor),
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Position: widget.RowLayoutPositionStart,
				}),
				widget.WidgetOpts.MinSize(250, 0), // 固定幅を設定
			),
		}
	}

	mainTextContainer := widget.NewContainer(mainTextContainerOpts...)

	mainText := widget.NewText(
		widget.TextOpts.Text(text, &res.Text.Face, textColor),
		widget.TextOpts.Position(widget.TextPositionStart, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)
	mainTextContainer.AddChild(mainText)
	container.AddChild(mainTextContainer)

	// 右側: 追加ラベル群（固定幅で配置、背景色なし）
	for _, label := range additionalLabels {
		labelContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Position: widget.RowLayoutPositionEnd,
				}),
				widget.WidgetOpts.MinSize(80, 0), // 追加ラベルも固定幅
			),
		)

		labelText := widget.NewText(
			widget.TextOpts.Text(label, &res.Text.Face, textColor),
			widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionEnd,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
				}),
			),
		)
		labelContainer.AddChild(labelText)
		container.AddChild(labelContainer)
	}

	return container
}

// NewFragmentText は色付きログフラグメント専用のテキストを作成する（文字数分だけの幅）
func NewFragmentText(text string, textColor color.RGBA, res *resources.UIResources) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(text, &res.Text.Face, textColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: false, // 横幅を伸ばさない
			}),
			// MinSizeは指定しない（テキストの自然な幅を使用）
		),
	)
}

// window ================

// NewSmallWindow は小さなウィンドウを作成する
func NewSmallWindow(title *widget.Container, content *widget.Container, opts ...widget.WindowOpt) *widget.Window {
	// デフォルトのオプション
	defaultOpts := []widget.WindowOpt{
		widget.WindowOpts.Contents(content),
		widget.WindowOpts.TitleBar(title, 25),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.CLICK_OUT),
		widget.WindowOpts.Draggable(),
		widget.WindowOpts.Resizeable(),
		widget.WindowOpts.MinSize(200, 200),
		widget.WindowOpts.MaxSize(650, 550),
	}
	allOpts := append(defaultOpts, opts...)

	return widget.NewWindow(allOpts...)
}

// opts ================

// BaseRowLayoutOpts は基本的な行レイアウトオプションを返す
func BaseRowLayoutOpts() []widget.RowLayoutOpt {
	return []widget.RowLayoutOpt{
		widget.RowLayoutOpts.Spacing(4),
		widget.RowLayoutOpts.Padding(&widget.Insets{
			Top:    10,
			Bottom: 10,
			Left:   4,
			Right:  4,
		}),
	}
}
