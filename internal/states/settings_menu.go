package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// SettingsMenuState はグローバル設定を変更するゲームステート。
// メインメニューから push される。現状は設定項目が無く、将来の設定（音量など）を追加する土台。
type SettingsMenuState struct {
	es.BaseState[w.World]
	menuMount *hooks.Mount[settingsMenuProps]
	widget    *ebitenui.UI
}

func (st SettingsMenuState) String() string {
	return "SettingsMenu"
}

// State interface ================

var _ es.State[w.World] = &SettingsMenuState{}
var _ es.ActionHandler[w.World] = &SettingsMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *SettingsMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *SettingsMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う。メインメニューの上に重なるためワールドは操作しない
func (st *SettingsMenuState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[settingsMenuProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *SettingsMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *SettingsMenuState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	st.menuMount.SetProps(st.fetchProps(world))
	props := st.menuMount.GetProps()
	hooks.UseTabMenu(st.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	if st.menuMount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *SettingsMenuState) Draw(world w.World, screen *ebiten.Image) error {
	bgImage, err := loadBackgroundImage(world, "title1")
	if err != nil {
		return err
	}
	screen.DrawImage(bgImage, nil)

	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *SettingsMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *SettingsMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(), nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("settingsMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// settingsMenuProps は設定メニューの表示に必要なプロパティを保持する
type settingsMenuProps struct {
	Items []settingsMenuItem
}

// settingsItemKind は設定項目の種類を表す
type settingsItemKind int

const (
	// settingsItemLanguage は言語を設定する項目を表す
	settingsItemLanguage settingsItemKind = iota
	// settingsItemBack は前の画面へ戻る項目を表す
	settingsItemBack
)

// settingsMenuItem は設定メニューの1項目を表す
type settingsMenuItem struct {
	Kind  settingsItemKind
	Label string
	Value string // 現在値の表示。値を持たない項目は空
}

func (st *SettingsMenuState) fetchProps(world w.World) settingsMenuProps {
	return settingsMenuProps{
		Items: []settingsMenuItem{
			{Kind: settingsItemLanguage, Label: "言語", Value: currentLanguageLabel(world.Config.User.Language)},
			{Kind: settingsItemBack, Label: "戻る"},
		},
	}
}

// focusedItem は現在カーソルが当たっている項目を返す
func (st *SettingsMenuState) focusedItem() (settingsMenuItem, bool) {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")
	if !ok || menuState.ItemIndex < 0 || menuState.ItemIndex >= len(props.Items) {
		return settingsMenuItem{}, false
	}
	return props.Items[menuState.ItemIndex], true
}

func (st *SettingsMenuState) handleSelection() es.Transition[w.World] {
	item, ok := st.focusedItem()
	if !ok {
		return es.Transition[w.World]{Type: es.TransNone}
	}
	switch item.Kind {
	case settingsItemLanguage:
		// Enter で言語選択のモーダルを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewLanguageMenuState}}
	case settingsItemBack:
		return es.Transition[w.World]{Type: es.TransPop}
	}
	return es.Transition[w.World]{Type: es.TransNone}
}

// ================
// 言語プリセット
// ================

// language は選択できる表示言語を表す
type language struct {
	Code  string // 言語コード（"ja" / "en"）
	Label string // 表示名（"日本語" / "English"）
}

// languagePresets は選択できる言語の一覧を保持する
var languagePresets = []language{
	{Code: "ja", Label: "日本語"},
	{Code: "en", Label: "English"},
}

// currentLanguageLabel は言語コードに対応する表示名を返す。一覧に無ければコードをそのまま返す
func currentLanguageLabel(code string) string {
	for _, l := range languagePresets {
		if l.Code == code {
			return l.Label
		}
	}
	return code
}

// NewLanguageMenuState は言語選択のモーダルを作成する。
// 選択した言語をユーザー設定に保存して設定画面へ戻る。実際の表示言語の切り替えは未実装。
func NewLanguageMenuState() (es.State[w.World], error) {
	messageState := &MessageState{}
	messageData := messagedata.NewSystemMessage("言語")
	for _, l := range languagePresets {
		messageData.WithChoice(l.Label, func(world w.World) error {
			world.Config.User.Language = l.Code
			if err := world.Config.SaveUserConfig(); err != nil {
				logger.New(logger.CategorySave).Warn("言語設定の保存に失敗しました", "error", err)
			}
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})
	}
	messageState.messageData = messageData
	return messageState, nil
}

// ================
// buildUI
// ================

func (st *SettingsMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")

	// 項目リストの描画は tabmenu.View に任せる。値は AdditionalLabels で右側に表示する
	items := make([]tabmenu.Item, len(props.Items))
	for i, item := range props.Items {
		it := tabmenu.Item{ID: item.Label, Label: item.Label}
		if item.Value != "" {
			// 現在値を右側に表示する。変更は Enter で開くモーダルから行う
			it.AdditionalLabels = []string{item.Value}
		}
		items[i] = it
	}
	view := tabmenu.NewView(tabmenu.Config{
		Tabs: []tabmenu.TabItem{{ID: "settings", Items: items}},
	}, world)
	view.SetState(tabmenu.ViewState{ItemIndex: menuState.ItemIndex})
	listContainer := view.BuildUI()

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	menuContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)

	titleText := widget.NewText(
		widget.TextOpts.Text("設定", &res.Text.BodyFace, theme.TextPrimary),
	)
	menuContainer.AddChild(titleText)
	menuContainer.AddChild(listContainer)

	rootContainer.AddChild(menuContainer)
	return &ebitenui.UI{Container: rootContainer}
}
