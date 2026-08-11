package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/screenui"
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

// ChoiceProps は選択メニューの表示スナップショット
type ChoiceProps struct {
	Title   string
	Choices []Choice
}

// ChoiceMenuState は本文と選択肢だけの単純メニュー。ダンジョンメニュー・デバッグメニュー・
// 交流メニュー・言語選択のように「選んで実行する」画面を1つの実装で担う。
// 選択肢は provide で毎フレーム解決するため、静的な画面は固定の選択肢を返せばよい
type ChoiceMenuState struct {
	es.BaseState[w.World]
	provide func(world w.World) (title string, choices []Choice)
	screen  *menuloop.Screen[ChoiceProps]
}

var _ es.State[w.World] = &ChoiceMenuState{}

// NewChoiceMenu は選択肢を返す provide を受け取り選択メニューを作る
func NewChoiceMenu(provide func(world w.World) (string, []Choice)) *ChoiceMenuState {
	return &ChoiceMenuState{provide: provide}
}

// OnStart はステートが開始される際に呼ばれる
func (st *ChoiceMenuState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[ChoiceProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *ChoiceMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *ChoiceMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
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
		return es.Transition[w.World]{}, fmt.Errorf("choiceMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// Fetch は現在の選択肢を表示スナップショットへ射影する
func (st *ChoiceMenuState) Fetch(world w.World) ChoiceProps {
	title, choices := st.provide(world)
	return ChoiceProps{Title: title, Choices: choices}
}

// Menu は単一タブの選択リストとして構成を返す。見出し行はカーソルを飛ばし、多い画面はページ送りする
func (st *ChoiceMenuState) Menu(props ChoiceProps) menuloop.MenuConfig {
	skips := make([]bool, len(props.Choices))
	for i, c := range props.Choices {
		skips[i] = c.Header
	}
	return menuloop.MenuConfig{Key: "choice", TabCount: 1, ItemCounts: []int{len(props.Choices)}, ItemsPerPage: menuItemsPerPage, Skips: [][]bool{skips}}
}

// View は選択肢の1カラム一覧を中央パネルに組む純粋描画。メインメニューやセーブロードと同じ簡易メニューの
// 見た目に揃え、エントリ数相応の大きさに縮む。多いときはページ送りしてはみ出さない
func (st *ChoiceMenuState) View(world w.World, props ChoiceProps, cursor menuloop.Selection, res resources.UIResources) *ebitenui.UI {
	rows := make([]menuRow, len(props.Choices))
	for i, c := range props.Choices {
		rows[i] = menuRow{Cells: styled.TextCells(c.Label), Header: c.Header}
	}
	// 単一タブのコマンドメニューなので行間を空け、ページ表示は複数ページのときだけ出す。
	// メインメニューと先頭位置・行間を揃える
	list := renderMenuList(cursor.ItemIndex, rows, []int{menuRowWidth}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{Spaced: true}, res)
	return screenui.NewPanelScreen(res, props.Title, list, menuNavHint(world, false))
}

// pushChoice は指定ファクトリの state を push する Choice.Run を返す。選択メニューの共通部品
func pushChoice(factory es.StateFactory[w.World]) func(w.World) (es.Transition[w.World], error) {
	return func(_ w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{factory}}, nil
	}
}

// popAfter は fn を実行して閉じる Choice.Run を返す
func popAfter(fn func(w.World) error) func(w.World) (es.Transition[w.World], error) {
	return func(world w.World) (es.Transition[w.World], error) {
		if err := fn(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
}

// stayAfter は fn を実行してメニューに留まる Choice.Run を返す。敵やPropの連続スポーンに使う
func stayAfter(fn func(w.World) error) func(w.World) (es.Transition[w.World], error) {
	return func(world w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransNone}, fn(world)
	}
}

// pushMessage は指定メッセージ画面を push する Choice.Run を返す
func pushMessage(md *messagedata.MessageData) func(w.World) (es.Transition[w.World], error) {
	return pushChoice(func() (es.State[w.World], error) { return NewMessageState(md) })
}
