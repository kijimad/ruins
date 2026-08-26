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
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// tabID は収納メニューのタブ識別子
type tabID string

// 収納メニューのタブID。タブ定義・転送処理で参照する。定義型にして任意文字列の混入を防ぐ
const (
	tabIDRetrieve tabID = "retrieve"
	tabIDStore    tabID = "store"
)

// StorageMenuState は収納メニューのゲームステート
type StorageMenuState struct {
	es.BaseState[w.World]
	storageEntity ecs.Entity
	detail        overlay.Detail // 詳細モーダル。overlay として Screen に登録する
	screen        *menuloop.Screen[StorageProps]
}

// State interface ================

var _ es.State[w.World] = &StorageMenuState{}
var _ menuloop.UIView[StorageProps] = &StorageMenuState{}
var _ menuloop.KeyBindings = &StorageMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *StorageMenuState) OnStart(_ w.World) error {
	st.detail = overlay.NewEntityDetail(st.selectedEntity)
	st.screen = menuloop.NewScreen[StorageProps](st, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *StorageMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 収納の出し入れで所持品が変わると WeightDirty が立つ。再計算を回して総重量表示を更新する
	if err := runUpdaters(world, &gs.WeightDirtySystem{}); err != nil {
		return es.Transition[w.World]{}, err
	}
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *StorageMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// KeyBindings は x の詳細表示を共通入力に足す
func (st *StorageMenuState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// DoAction はActionを実行する
func (st *StorageMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuSelect:
		if err := st.executeTransfer(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	default:
		return es.Transition[w.World]{}, fmt.Errorf("storageMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// StorageProps は画面の表示 props。menuloop.Screen の型引数として渡す
type StorageProps struct {
	Tabs []storageTabData
}

type storageTabData struct {
	ID    tabID
	Label string
	Items []itemRowData
}

// Fetch は世界から表示 props を構築する。menuloop.Model の Model 部にあたる。
// 収納メニューはプレイヤーの操作でしか開かないので、プレイヤー不在は不変条件違反として返す
func (st *StorageMenuState) Fetch(world w.World) (StorageProps, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return StorageProps{}, err
	}
	return StorageProps{
		Tabs: []storageTabData{
			{ID: tabIDRetrieve, Label: query.T(world, "Retrieve"), Items: st.toStorageItemData(world, query.StorageStacks(world, st.storageEntity))},
			{ID: tabIDStore, Label: query.T(world, "Store"), Items: st.toStorageItemData(world, query.BackpackStacks(world, player))},
		},
	}, nil
}

// Menu は一覧の構成を返す。menuloop.Model の Menu 部にあたる
func (st *StorageMenuState) Menu(props StorageProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menuloop.MenuConfig{Key: "storage", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

func (st *StorageMenuState) toStorageItemData(world w.World, stacks []query.Stack) []itemRowData {
	// 1スタック1行。重量は束の総量、個数は束の大きさを出す
	items := make([]itemRowData, len(stacks))
	for i, stack := range stacks {
		rep := stack.Rep
		total := query.GetEntityWeight(world, rep) * consts.Milligram(stack.Count)
		items[i] = itemRowData{
			Entity: rep,
			Name:   query.GetEntityName(rep, world),
			Weight: total.KgString(),
			Count:  stack.Count,
		}
	}
	return items
}

// ================
// アクション実行
// ================

func (st *StorageMenuState) executeTransfer(world w.World) error {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	tabIndex := cursor.TabIndex
	itemIndex := cursor.ItemIndex

	if tabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[tabIndex]
	if len(tab.Items) == 0 || itemIndex >= len(tab.Items) {
		return nil
	}

	item := tab.Items[itemIndex]

	switch tab.ID {
	case tabIDRetrieve:
		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		if _, err := lifecycle.MoveStackToBackpack(world, item.Entity, playerEntity); err != nil {
			return err
		}
	case tabIDStore:
		// 容量判定が束の合計重量を要するため、ここだけ実体列を先に束ねて可否を見てから移す
		members := query.StackMembers(world, item.Entity)
		if !query.CanAddStackToStorage(world, st.storageEntity, members) {
			return nil
		}
		if _, err := lifecycle.MoveMembersToStorage(world, members, st.storageEntity); err != nil {
			return err
		}
	}

	return nil
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menuloop.Model の View 部にあたる
func (st *StorageMenuState) View(world w.World, props StorageProps, cursor menuloop.Selection, res resources.UIResources) *ebitenui.UI {
	// カテゴリをタブ帯に寄せ、本体は1カラムの一覧にする。性能は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	return menuframe.NewTabScreen(res, menuframe.TabScreen{
		TabLabels: labels,
		TabIndex:  cursor.TabIndex,
		Content:   st.buildActiveListContainer(world, props, cursor.TabIndex, cursor.ItemIndex, res),
		Footer:    keybind.HelpHint(world),
	})
}

// ViewUI は View の internal/ui 版。カテゴリタブとアイコン付きアイテム一覧を自前 UI で組む。
// 詳細モーダルは ScreenRenderer として Screen が本体の上へ重ねる。
func (st *StorageMenuState) ViewUI(world w.World, props StorageProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	content := st.buildActiveListUI(world, props, cursor.TabIndex, cursor.ItemIndex, res)
	return buildTabScreenUI(world, res, "", labels, cursor.TabIndex, content, keybind.HelpHint(world))
}

// buildActiveListUI は buildActiveListContainer の internal/ui 版。アイコン付きの1カラム一覧を返す。
func (st *StorageMenuState) buildActiveListUI(world w.World, props StorageProps, tabIndex, itemIndex int, res resources.UIResources) []ui.Widget {
	if tabIndex >= len(props.Tabs) {
		return nil
	}
	currentTab := props.Tabs[tabIndex]
	columnWidths, aligns := itemMenuColumns(260, menuColumn{Width: 80, Align: styled.AlignRight})
	rows := make([]menuRow, len(currentTab.Items))
	for i, it := range currentTab.Items {
		rows[i] = itemMenuRow(world, it.Entity, it.Count, it.Weight)
	}
	return renderMenuListUI(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No items")}, res.Text.BodyFace)
}

// selectedEntity は現在カーソルが当たっているアイテムのエンティティを返す
func (st *StorageMenuState) selectedEntity() (ecs.Entity, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return ecs.Entity{}, false
	}
	items := props.Tabs[cursor.TabIndex].Items
	if cursor.ItemIndex >= len(items) {
		return ecs.Entity{}, false
	}
	return items[cursor.ItemIndex].Entity, true
}

func (st *StorageMenuState) buildActiveListContainer(world w.World, props StorageProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	if tabIndex >= len(props.Tabs) {
		return styled.NewVerticalContainer()
	}

	currentTab := props.Tabs[tabIndex]
	columnWidths, aligns := itemMenuColumns(260, menuColumn{Width: 80, Align: styled.AlignRight})
	rows := make([]menuRow, len(currentTab.Items))
	for i, it := range currentTab.Items {
		rows[i] = itemMenuRow(world, it.Entity, it.Count, it.Weight)
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No items")}, res)
}
