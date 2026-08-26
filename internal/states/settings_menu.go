package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
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
func (st *SettingsMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(world), nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("settingsMenu: unsupported action: %s", action)
	}
}

// cycleFocused はカーソル上の項目の値を次へ循環させる。値を持たない項目では何もしない
func (st *SettingsMenuState) cycleFocused(world w.World) {
	item, ok := st.focusedItem()
	if !ok {
		return
	}
	if item.Kind == settingsItemLanguage {
		cycleLanguage(world)
	}
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

func (st *SettingsMenuState) handleSelection(world w.World) es.Transition[w.World] {
	item, ok := st.focusedItem()
	if !ok {
		return es.Transition[w.World]{Type: es.TransNone}
	}
	if item.Kind == settingsItemBack {
		return es.Transition[w.World]{Type: es.TransPop}
	}
	// 値を持つ項目は Enter で次の値へ循環する
	st.cycleFocused(world)
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

// cycleLanguage は表示言語を次へ循環させ、シングルトンへ即時反映してユーザー設定へ保存する。
func cycleLanguage(world w.World) {
	langs := i18n.SupportedLangs()
	if len(langs) == 0 {
		return
	}
	current := query.GetUserSettings(world).Language
	idx := 0
	for i, l := range langs {
		if l.Code == current {
			idx = i
			break
		}
	}
	next := langs[(idx+1)%len(langs)].Code
	applyLanguage(world, next)
}

// applyLanguage は表示言語を code に切り替える。
// Fetch が毎フレーム query.T 経由で引き直すので、再起動なしで表示が変わる。
// UserSettings は world 初期化で常に登録されるため nil を想定しない。Fetch も同じ前提で読む。
func applyLanguage(world w.World, code string) {
	query.GetUserSettings(world).Language = code
	world.Resources.Config.User.Language = code
	if err := world.Resources.Config.SaveUserConfig(); err != nil {
		logger.New(logger.CategorySave).Warn("failed to save language setting", "error", err)
	}
}

// ================
// View
// ================

// ViewUI は View の internal/ui 版。設定項目の2列表を中央パネルに自前 UI で組む。
func (st *SettingsMenuState) ViewUI(world w.World, props SettingsMenuProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	rows := make([]menuRow, len(props.Items))
	for i, item := range props.Items {
		rows[i] = menuRow{Cells: styled.TextCells(item.Label, item.Value)}
	}
	content, pager := renderMenuListUI(cursor.ItemIndex, rows, []int{240, 100}, []styled.TextAlign{styled.AlignLeft, styled.AlignRight}, menuListOpts{Spaced: true}, res)
	return buildPanelScreenUI(world, res, query.T(world, "Settings"), content, keybind.HelpHint(world), pager)
}
