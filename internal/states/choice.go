package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
)

// Choice は選択メニューの1項目。Run が選択時の実行で、戻り値のステート遷移で
// 画面を閉じる、別画面へ進む、などを表す。Header が真の行はカーソルが止まらない見出し
type Choice struct {
	Label  string
	Run    func(world w.World) (es.Transition[w.World], error)
	Header bool
}

// choiceProps は選択メニューの表示スナップショット
type choiceProps struct {
	Title   string
	Choices []Choice
}

// ChoiceMenuState は本文と選択肢だけの単純メニュー。ダンジョンメニュー・デバッグメニュー・
// 交流メニュー・言語選択のように「選んで実行する」画面を1つの実装で担う。
// 選択肢は provide で毎フレーム解決するため、静的な画面は固定の選択肢を返せばよい
type ChoiceMenuState struct {
	es.BaseState[w.World]
	provide func(world w.World) (title string, choices []Choice)
	screen  Screen[choiceProps]
}

var (
	_ es.State[w.World]         = &ChoiceMenuState{}
	_ es.ActionHandler[w.World] = &ChoiceMenuState{}
	_ Configurable              = &ChoiceMenuState{}
)

// NewChoiceMenu は選択肢を返す provide を受け取り選択メニューを作る
func NewChoiceMenu(provide func(world w.World) (string, []Choice)) *ChoiceMenuState {
	return &ChoiceMenuState{provide: provide}
}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *ChoiceMenuState) StateConfig() StateConfig { return StateConfig{BlurBackground: false} }

// OnPause はステートが一時停止される際に呼ばれる
func (st *ChoiceMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *ChoiceMenuState) OnResume(_ w.World) error { return nil }

// OnStop はステートが停止される際に呼ばれる
func (st *ChoiceMenuState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *ChoiceMenuState) OnStart(_ w.World) error {
	st.screen = NewScreen[choiceProps]()
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *ChoiceMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world, st)
}

// Draw はゲームステートの描画処理を行う
func (st *ChoiceMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// HandleInput はキー入力を Action に変換する
func (st *ChoiceMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction は Action を実行する。選択で現在行の Run を呼び、その遷移を返す
func (st *ChoiceMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		choices := st.screen.Props().Choices
		i := st.screen.Selection().ItemIndex
		if i < 0 || i >= len(choices) || choices[i].Header || choices[i].Run == nil {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		return choices[i].Run(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatch で処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("choiceMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// fetch は現在の選択肢を表示スナップショットへ射影する
func (st *ChoiceMenuState) fetch(world w.World) choiceProps {
	title, choices := st.provide(world)
	return choiceProps{Title: title, Choices: choices}
}

// menu は単一タブの選択リストとして構成を返す。見出し行はカーソルを飛ばし、多い画面はページ送りする
func (st *ChoiceMenuState) menu(props choiceProps) MenuConfig {
	skips := make([]bool, len(props.Choices))
	for i, c := range props.Choices {
		skips[i] = c.Header
	}
	return MenuConfig{Key: "choice", TabCount: 1, ItemCounts: []int{len(props.Choices)}, ItemsPerPage: menuItemsPerPage, Skips: [][]bool{skips}}
}

// view は選択肢の1カラム一覧を中央パネルに組む純粋描画。メインメニューやセーブロードと同じ簡易メニューの
// 見た目に揃え、エントリ数相応の大きさに縮む。多いときはページ送りしてはみ出さない
func (st *ChoiceMenuState) view(_ w.World, props choiceProps, sel Selection, res resources.UIResources) *ebitenui.UI {
	list := styled.NewVerticalContainer()
	pg := pagination.New(sel.ItemIndex, len(props.Choices), menuItemsPerPage)
	// ページ表示は複数ページのときだけ出す。単一ページの簡易メニューは余計な行を置かず、
	// メインメニューと先頭位置・行間を揃える
	if pg.IsEnabled() {
		list.AddChild(newPageIndicator(pg, res))
	}

	columnWidths := []int{menuRowWidth}
	aligns := []styled.TextAlign{styled.AlignLeft}
	table := newMenuListTable(columnWidths, res)
	visible := pagination.VisibleEntries(props.Choices, pg)
	for _, entry := range visible {
		if entry.Item.Header {
			styled.NewTableHeaderRow(table, columnWidths, []string{entry.Item.Label}, res)
			continue
		}
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, columnWidths, []string{entry.Item.Label}, aligns, &isSelected, res)
	}
	// 複数ページの画面は各ページを1ページ件数ぶんの行に埋め、ページを繰ってもパネルの高さが変わらないようにする。
	// 単一ページの画面は埋めず、エントリ数相応の大きさに縮む
	if len(props.Choices) > menuItemsPerPage {
		for i := len(visible); i < menuItemsPerPage; i++ {
			notSelected := false
			styled.NewTableRow(table, columnWidths, []string{" "}, aligns, &notSelected, res)
		}
	}
	list.AddChild(table)

	return newPanelScreenUI(res, props.Title, list, menuNavHint(false))
}
