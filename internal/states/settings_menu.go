package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// SettingsMenuState はグローバル設定を変更するゲームステート。
// メインメニューから push される。現状は設定項目が無く、将来の設定（音量など）を追加する土台。
type SettingsMenuState struct {
	es.BaseState[w.World]
	screen *menuloop.Screen[SettingsMenuProps]
}

// State interface ================

var _ es.State[w.World] = &SettingsMenuState{}

// OnStart はステート開始時の処理を行う。メインメニューの上に重なるためワールドは操作しない
func (st *SettingsMenuState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[SettingsMenuProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *SettingsMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はスクリーンに描画する
func (st *SettingsMenuState) Draw(world w.World, screen *ebiten.Image) error {
	bgImage, err := loadBackgroundImage(world, "title1")
	if err != nil {
		return err
	}
	screen.DrawImage(bgImage, nil)

	st.screen.Draw(screen)
	return nil
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
		return es.Transition[w.World]{}, fmt.Errorf("settingsMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// SettingsMenuProps は設定メニューの表示に必要なプロパティを保持する
type SettingsMenuProps struct {
	Items []settingsMenuItem
}

// settingsItemKind は設定項目の種類を表す
type settingsItemKind string

const (
	// settingsItemLanguage は言語を設定する項目を表す
	settingsItemLanguage settingsItemKind = "language"
	// settingsItemBack は前の画面へ戻る項目を表す
	settingsItemBack settingsItemKind = "back"
)

// settingsMenuItem は設定メニューの1項目を表す
type settingsMenuItem struct {
	Kind  settingsItemKind
	Label string
	Value string // 現在値の表示。値を持たない項目は空
}

// Fetch は世界から表示 props を構築する。menuloop.Model の Model 部にあたる
func (st *SettingsMenuState) Fetch(world w.World) (SettingsMenuProps, error) {
	return SettingsMenuProps{
		Items: []settingsMenuItem{
			{Kind: settingsItemLanguage, Label: query.T(world, "Language"), Value: query.T(world, currentLanguageLabel(query.GetUserSettings(world).Language))},
			{Kind: settingsItemBack, Label: query.T(world, "Back")},
		},
	}, nil
}

// Menu は一覧の構成を返す。menuloop.Model の Menu 部にあたる
func (st *SettingsMenuState) Menu(props SettingsMenuProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "menu", TabCount: 1, ItemCounts: []int{len(props.Items)}}
}

// focusedItem は現在カーソルが当たっている項目を返す
func (st *SettingsMenuState) focusedItem() (settingsMenuItem, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(props.Items) {
		return settingsMenuItem{}, false
	}
	return props.Items[cursor.ItemIndex], true
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

// currentLanguageLabel は言語コードに対応する表示名の msgid を返す。対応一覧に無ければコードをそのまま返す。
func currentLanguageLabel(code string) string {
	for _, l := range i18n.SupportedLangs() {
		if l.Code == code {
			return l.Label
		}
	}
	return code
}

// NewLanguageMenuState は言語選択のモーダルを作成する。
// 選択した言語をシングルトンへ即時反映し、ユーザー設定へ保存して設定画面へ戻る。
func NewLanguageMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(languageChoices), nil
}

// languageChoices は選択できる表示言語を選択肢にする。選ぶとシングルトンへ反映し設定を保存して閉じる
func languageChoices(world w.World) (string, []Choice) {
	langs := i18n.SupportedLangs()
	choices := make([]Choice, 0, len(langs))
	for _, l := range langs {
		choices = append(choices, Choice{Label: query.T(world, l.Label), Run: func(world w.World) (es.Transition[w.World], error) {
			// シングルトンの設定言語を書き換える。Fetch が毎フレーム query.T 経由で引き直すので、再起動なしで表示が変わる。
			if s := query.GetUserSettings(world); s != nil {
				s.Language = l.Code
			}
			world.Config.User.Language = l.Code
			if err := world.Config.SaveUserConfig(); err != nil {
				logger.New(logger.CategorySave).Warn("failed to save language setting", "error", err)
			}
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}})
	}
	return "", choices
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menuloop.Model の View 部にあたる
func (st *SettingsMenuState) View(world w.World, props SettingsMenuProps, cursor menuloop.Selection, res resources.UIResources) *ebitenui.UI {
	// 項目リストは他メニューと同じテーブル描画に揃える。現在値は右列に表示し、変更は Enter で開くモーダルから行う
	rows := make([]menuRow, len(props.Items))
	for i, item := range props.Items {
		rows[i] = menuRow{Cells: styled.TextCells(item.Label, item.Value)}
	}
	table := renderMenuList(cursor.ItemIndex, rows, []int{240, 100}, []styled.TextAlign{styled.AlignLeft, styled.AlignRight}, menuListOpts{Spaced: true}, res)

	menuContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.Image),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)

	titleText := widget.NewText(
		widget.TextOpts.Text(query.T(world, "Settings"), &res.Text.BodyFace, theme.TextPrimary),
	)
	menuContainer.AddChild(titleText)
	menuContainer.AddChild(table)
	menuContainer.AddChild(styled.NewDescriptionText(menuNavHint(world, false), res))

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	rootContainer.AddChild(menuContainer)
	return &ebitenui.UI{Container: rootContainer}
}
