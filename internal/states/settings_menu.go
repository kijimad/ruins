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
	"github.com/kijimaD/ruins/internal/widgets/styled"
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
func (st *SettingsMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(), nil
	case inputmapper.ActionMenuLeft:
		st.changeLanguage(world, -1)
	case inputmapper.ActionMenuRight:
		st.changeLanguage(world, 1)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("settingsMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// settingsMenuProps は設定メニューのProps
type settingsMenuProps struct {
	Items []settingsMenuItem
}

// settingsItemKind は設定項目の種類
type settingsItemKind int

const (
	// settingsItemLanguage は言語設定項目
	settingsItemLanguage settingsItemKind = iota
	// settingsItemBack は前の画面へ戻る項目
	settingsItemBack
)

// settingsMenuItem は設定メニューの1項目
type settingsMenuItem struct {
	Kind  settingsItemKind
	Label string
	Value string // 現在値の表示。値を持たない項目は空
}

// display は項目の表示文字列を返す。値を持つ項目は左右カーソルで変更できることを示す
func (it settingsMenuItem) display() string {
	if it.Value == "" {
		return it.Label
	}
	return fmt.Sprintf("%s  ◀ %s ▶", it.Label, it.Value)
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
	if item.Kind == settingsItemBack {
		return es.Transition[w.World]{Type: es.TransPop}
	}
	return es.Transition[w.World]{Type: es.TransNone}
}

// changeLanguage は言語項目にカーソルがある場合に言語を切り替えて保存する。
// 実際の表示言語の切り替えは未実装で、設定値の保持のみを行う。
func (st *SettingsMenuState) changeLanguage(world w.World, dir int) {
	item, ok := st.focusedItem()
	if !ok || item.Kind != settingsItemLanguage {
		return
	}
	next := nextLanguage(world.Config.User.Language, dir)
	if err := applyLanguage(world, next); err != nil {
		logger.New(logger.CategorySave).Warn("言語設定の保存に失敗しました", "error", err)
	}
}

// ================
// 言語プリセット
// ================

// language は選択できる表示言語
type language struct {
	Code  string // 言語コード（"ja" / "en"）
	Label string // 表示名（"日本語" / "English"）
}

// languagePresets は選択できる言語の一覧
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

// nextLanguage は現在の言語コードから dir 方向（+1/-1）の次の言語を返す。
// 一覧の端では反対側へ循環する。現在値が一覧に無い場合は先頭を起点とする。
func nextLanguage(code string, dir int) language {
	idx := 0
	for i, l := range languagePresets {
		if l.Code == code {
			idx = i
			break
		}
	}
	n := len(languagePresets)
	idx = ((idx+dir)%n + n) % n
	return languagePresets[idx]
}

// applyLanguage は言語をユーザー設定に保存する。実際の表示切り替えは未実装。
func applyLanguage(world w.World, l language) error {
	world.Config.User.Language = l.Code
	return world.Config.SaveUserConfig()
}

// ================
// buildUI
// ================

func (st *SettingsMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")
	itemIndex := menuState.ItemIndex

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

	for i, item := range props.Items {
		isSelected := i == itemIndex
		itemWidget := styled.NewListItemText(item.display(), theme.TextSecondary, isSelected, res)
		menuContainer.AddChild(itemWidget)
	}

	rootContainer.AddChild(menuContainer)
	return &ebitenui.UI{Container: rootContainer}
}
