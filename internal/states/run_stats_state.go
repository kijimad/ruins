package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// run 統計の閲覧画面。character・inventory と同じ menuframe のタブ画面枠に、統計を
// Label と Value の2列テーブルで並べる。読み取り専用で、道中の統計閲覧と死の結果画面を
// 見出しと閉じ方だけ変えて共用する。値の組み立ては runStatsItems に1本化する。

const runStatsMenuKey = "run_stats"

// RunStatsState は run 統計をテーブルで見る読み取り専用画面
type RunStatsState struct {
	es.BaseState[w.World]
	headerMsgid  string                 // 見出しの msgid。統計は "Statistics"、結果は "You died."
	exit         es.Transition[w.World] // Cancel で抜ける先。統計は Pop、結果はメインメニューへ Replace
	exitOnSelect bool                   // Select でも exit へ抜けるか。結果画面は任意キーで戻せるよう真にする
	screen       *menuloop.Screen[RunStatsProps]
}

// RunStatsProps は統計画面の表示 props
type RunStatsProps struct {
	Items []statusItemData
}

var _ es.State[w.World] = &RunStatsState{}

// NewRunStatsState は道中で run 統計を見る画面を作る。常時メニューから開き、閉じると元へ戻る
func NewRunStatsState() (es.State[w.World], error) {
	return &RunStatsState{
		headerMsgid: "Statistics",
		exit:        es.Transition[w.World]{Type: es.TransPop},
	}, nil
}

// NewRunResultState は run の死の結果画面を作る。統計を同じテーブルで見せ、任意キーでメインメニューへ戻す
func NewRunResultState() (es.State[w.World], error) {
	return &RunStatsState{
		headerMsgid:  "You died.",
		exit:         es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState}},
		exitOnSelect: true,
	}, nil
}

// OnStart は Screen を組み立てる。overlay は持たない読み取り専用画面
func (st *RunStatsState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[RunStatsProps](st)
	return nil
}

// Update はステートの更新処理を Screen へ委譲する
func (st *RunStatsState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はステートの描画を Screen へ委譲する
func (st *RunStatsState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// DoAction は閲覧中の Action を処理する。読み取り専用なので Cancel と Select だけ扱う
func (st *RunStatsState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return st.exit, nil
	case inputmapper.ActionMenuSelect:
		if st.exitOnSelect {
			return st.exit, nil
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("unknown action: %s", action)
	}
}

// Fetch は表示する統計行を組む
func (st *RunStatsState) Fetch(world w.World) (RunStatsProps, error) {
	return RunStatsProps{Items: runStatsItems(world)}, nil
}

// Menu は単一タブの読み取り専用構成を返す。見出し行が無いのでスキップは不要
func (st *RunStatsState) Menu(props RunStatsProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: runStatsMenuKey, TabCount: 1, ItemCounts: []int{len(props.Items)}}
}

// View は見出しと統計テーブルを menuframe のタブ画面枠へ組む。ラベルの訳のみ world から引く
func (st *RunStatsState) View(world w.World, props RunStatsProps, cursor menuloop.Selection, res resources.UIResources) *ebitenui.UI {
	content := buildStatsTable(world, props.Items, cursor.ItemIndex, res)
	return menuframe.NewTabScreen(res, menuframe.TabScreen{
		Header:  query.T(world, st.headerMsgid),
		Content: content,
		Footer:  keybind.HelpHint(world),
	})
}

// buildStatsTable は統計を Label と Value の2列テーブルに組む。ラベルが長いので
// character 情報タブより広い列幅を取り、値は右寄せの数値にする
func buildStatsTable(world w.World, items []statusItemData, itemIndex int, res resources.UIResources) *widget.Container {
	columnWidths := []int{180, 90}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(items))
	for i, it := range items {
		rows[i] = menuRow{Cells: styled.TextCells(it.Label, it.Value)}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{
		AlwaysIndicator: true,
		EmptyText:       query.T(world, "No entries"),
		ItemsPerPage:    menuframe.ListCapacity(res, true, false),
	}, res)
}

// runStatsItems は run 統計をテーブル行に組む。結果画面と道中の統計画面で共通。
// Label と Value の2列で、値は右寄せの数値。到達度は経過ターンで示す
func runStatsItems(world w.World) []statusItemData {
	days, turns, kills, items, sales := runStatsFields(world)
	return []statusItemData{
		{Label: query.T(world, "Days"), Value: fmt.Sprintf("%d", days)},
		{Label: query.T(world, "Turns"), Value: fmt.Sprintf("%d", turns)},
		{Label: query.T(world, "Enemies killed"), Value: fmt.Sprintf("%d", kills)},
		{Label: query.T(world, "Items scavenged"), Value: fmt.Sprintf("%d", items)},
		{Label: query.T(world, "Sales"), Value: fmt.Sprintf("%d", sales)},
	}
}

// runStatsFields は現在の統計を返す。撃破・漁り・売上は RunStats、日数・ターンは GameTime から引く
func runStatsFields(world w.World) (days, turns, kills, items int, sales consts.Currency) {
	if s := query.GetRunStats(world); s != nil {
		kills = s.EnemiesKilled
		items = s.ItemsScavenged
		sales = s.SalesTotal
	}
	if gt := query.GetGameTime(world); gt != nil {
		days = gt.GetDayNumber()
		turns = int(gt.TotalTurns)
	}
	return
}
