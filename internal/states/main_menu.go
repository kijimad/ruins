package states

import (
	"fmt"
	"image"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	w "github.com/kijimaD/ruins/internal/world"
)

// MainMenuState は新しいメニューコンポーネントを使用するメインメニュー
type MainMenuState struct {
	es.BaseState[w.World]
	ui       *ebitenui.UI
	menuView *tabmenu.View
}

func (st MainMenuState) String() string {
	return "MainMenu"
}

// State interface ================

var _ es.State[w.World] = &MainMenuState{}
var _ es.ActionHandler[w.World] = &MainMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MainMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MainMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *MainMenuState) OnStart(world w.World) error {
	// ワールドをクリアする。前のゲーム状態を削除する
	world.Manager.DeleteAllEntities()

	st.initMenu(world)
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *MainMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *MainMenuState) Update(_ w.World) (es.Transition[w.World], error) {
	// メニューの更新（キーボード入力→Action変換は TabMenu 内部で実施）
	if err := st.menuView.Update(); err != nil {
		return es.Transition[w.World]{Type: es.TransNone}, err
	}

	if st.ui != nil {
		st.ui.Update()
	}

	// BaseStateの共通処理を使用
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *MainMenuState) Draw(world w.World, screen *ebiten.Image) error {
	// 背景画像を描画
	bgSheet := (*world.Resources.SpriteSheets)["bg"]
	bgSprite := bgSheet.Sprites["title1"]
	rect := image.Rect(
		bgSprite.X,
		bgSprite.Y,
		bgSprite.X+bgSprite.Width,
		bgSprite.Y+bgSprite.Height,
	)
	bgImage := bgSheet.Texture.Image.SubImage(rect).(*ebiten.Image)
	screen.DrawImage(bgImage, nil)

	st.ui.Draw(screen)
	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *MainMenuState) HandleInput() (inputmapper.ActionID, bool) {
	// 未使用
	return "", false
}

// DoAction はActionを実行する
func (st *MainMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel:
		// メインメニューでのキャンセルは終了
		return es.Transition[w.World]{Type: es.TransQuit}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// ================

// initMenu はメニューコンポーネントを初期化する
func (st *MainMenuState) initMenu(world w.World) {
	// タブ1つでTabMenuを使用
	tabs := []tabmenu.TabItem{
		{
			ID:    "main",
			Label: "",
			Items: []tabmenu.Item{
				{
					ID:       "town",
					Label:    "開始",
					UserData: es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewTownState()}},
				},
				{
					ID:       "load",
					Label:    "読込",
					UserData: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewLoadMenuState}},
				},
				{
					ID:       "exit",
					Label:    "終了",
					UserData: es.Transition[w.World]{Type: es.TransQuit},
				},
			},
		},
	}

	// TabMenuの設定
	config := tabmenu.Config{
		Tabs:             tabs,
		InitialTabIndex:  0,
		InitialItemIndex: 0,
		WrapNavigation:   true,
		ItemsPerPage:     10,
	}

	// コールバックの設定
	callbacks := tabmenu.Callbacks{
		OnSelectItem: func(_ int, _ int, _ tabmenu.TabItem, item tabmenu.Item) error {
			// 選択されたアイテムのUserDataからTransitionを取得
			if trans, ok := item.UserData.(es.Transition[w.World]); ok {
				st.SetTransition(trans)
			}
			return nil
		},
		OnCancel: func() {
			// Escapeキーが押された時の処理
			st.SetTransition(es.Transition[w.World]{Type: es.TransQuit})
		},
	}

	// View を作成（TabMenu + UIBuilder を統合）
	st.menuView = tabmenu.NewView(config, callbacks, world)
}

// initUI はUIを初期化する
func (st *MainMenuState) initUI(world w.World) *ebitenui.UI {
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// TabMenuView を使ってメニューUIを構築
	menuContainer := st.menuView.BuildUI()

	// バージョン表示テキストを作成
	versionInfo := []string{}
	if consts.AppVersion != "v0.0.0" {
		versionInfo = append(versionInfo, consts.AppVersion)
	}
	if consts.AppCommit != "0000000" {
		versionInfo = append(versionInfo, consts.AppCommit)
	}
	if consts.AppDate != "0000-00-00" {
		versionInfo = append(versionInfo, consts.AppDate)
	}
	versionText := widget.NewText(
		widget.TextOpts.Text(strings.Join(versionInfo, "\n"), &world.Resources.UIResources.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: &widget.Insets{
					Right:  20, // 画面右端から20ピクセル左に配置
					Bottom: 20, // 画面下端から20ピクセル上に配置
				},
			}),
		),
	)

	// ラッパーコンテナを作成(メニューの位置指定のため)
	wrapperContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding: &widget.Insets{
					Top: 400, // メニューを下寄りにする
				},
			}),
		),
	)

	// メニューコンテナをラッパーに追加
	wrapperContainer.AddChild(menuContainer)

	// メニュー、バージョンテキストをrootContainerに追加
	rootContainer.AddChild(wrapperContainer)
	rootContainer.AddChild(versionText)

	return &ebitenui.UI{Container: rootContainer}
}
