package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/ui"
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
	Tabs []statsTab
}

// statsTab は統計画面の1タブ。見出しと2列テーブルの行を持つ
type statsTab struct {
	Label string
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

// Fetch は表示するタブと統計行を組む。統計と環境の2タブを持つ
func (st *RunStatsState) Fetch(world w.World) (RunStatsProps, error) {
	return RunStatsProps{Tabs: []statsTab{
		{Label: query.T(world, "Summary"), Items: runStatsItems(world)},
		{Label: query.T(world, "Environment"), Items: environmentItems(world)},
	}}, nil
}

// Menu は読み取り専用のタブ構成を返す。見出し行が無いのでスキップは不要
func (st *RunStatsState) Menu(props RunStatsProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menuloop.MenuConfig{Key: runStatsMenuKey, TabCount: len(props.Tabs), ItemCounts: itemCounts}
}

// ViewUI は View の internal/ui 版。タブ帯つきのステータス表を自前 UI で組む。
func (st *RunStatsState) ViewUI(world w.World, props RunStatsProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	tabIndex := cursor.TabIndex
	if tabIndex >= len(props.Tabs) {
		tabIndex = 0
	}
	content := buildStatsTableUI(world, props.Tabs[tabIndex].Items, cursor.ItemIndex, res)
	return buildTabScreenUI(world, res, query.T(world, st.headerMsgid), labels, tabIndex, content, keybind.HelpHint(world))
}

// buildStatsTableUI は buildStatsTable の internal/ui 版。ラベル左・値右の2列表を行ウィジェット列で返す。
func buildStatsTableUI(world w.World, items []statusItemData, itemIndex int, res resources.UIResources) []ui.Widget {
	columnWidths := []int{180, 90}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(items))
	for i, it := range items {
		rows[i] = menuRow{Cells: styled.TextCells(it.Label, it.Value)}
	}
	return renderMenuListUI(itemIndex, rows, columnWidths, aligns, menuListOpts{
		AlwaysIndicator: true,
		EmptyText:       query.T(world, "No entries"),
		ItemsPerPage:    menuframe.ListCapacity(world, true, true),
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

// environmentItems は現在地の環境情報をテーブル行に組む。周囲気温・時間帯・季節。
// 周囲気温はプレイヤーの位置から引く。プレイヤーが居ない、または座標を持たないときは0にする
func environmentItems(world w.World) []statusItemData {
	gt := query.GetGameTime(world)
	if gt == nil {
		return nil
	}

	ambient := 0
	if player, err := query.GetPlayerEntity(world); err == nil && query.AliveHas(world, world.Components.GridElement, player) {
		g := world.Components.GridElement.Get(player)
		if temp, terr := systems.AmbientTemperatureAt(world, g.X, g.Y); terr == nil {
			ambient = temp
		}
	}

	return []statusItemData{
		{Label: query.T(world, "Ambient temperature"), Value: fmt.Sprintf("%d%s", ambient, consts.IconDegree)},
		{Label: query.T(world, "Time of day"), Value: query.T(world, gt.GetTimeOfDay().String())},
		{Label: query.T(world, "Season"), Value: query.T(world, gt.GetSeason().String())},
	}
}
