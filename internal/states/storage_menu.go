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
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
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
	// itemFilter は取り出し・投入の両タブに出す品を絞る述語。nil なら全許可。
	// キューブの燃料投入で、燃料だけを見せ貨物を隠す用途で使う
	itemFilter func(w.World, ecs.Entity) bool
	// showHeat は各行に熱量列を重量の左へ出すか。燃料メニューでだけ true にする
	showHeat bool
	// storeOnly は投入タブだけを出すか。燃料は入れたら即残数になり取り出さないので取り出しタブを隠す
	storeOnly bool
	// titleFunc は画面上部の見出しを世界から導く。nil なら見出し無し。ViewUI で毎フレーム呼ぶので
	// 燃料の残数など投入で変わる値を出すと即座に反映される
	titleFunc func(w.World) string
	detail    overlay.Detail // 詳細モーダル。overlay として Screen に登録する
	screen    *menuloop.Screen[StorageProps]
}

// StorageOption は StorageMenuState の任意設定
type StorageOption func(*StorageMenuState)

// WithItemFilter は両タブに出す品を述語で絞る。燃料投入のように扱う品目を限定する用途で使う
func WithItemFilter(pred func(w.World, ecs.Entity) bool) StorageOption {
	return func(st *StorageMenuState) { st.itemFilter = pred }
}

// WithHeatColumn は各行に熱量列を重量の左へ足す。燃料メニューで燃料性能を熱量で比べる用途で使う。
// 汎用の収納メニューには渡さず、この文脈でだけ列を増やす
func WithHeatColumn() StorageOption {
	return func(st *StorageMenuState) { st.showHeat = true }
}

// WithStoreOnly は取り出しタブを隠し投入タブだけにする。燃料は入れたら即残数になり取り出さないので、
// 取り出しの選択肢自体を無くす
func WithStoreOnly() StorageOption {
	return func(st *StorageMenuState) { st.storeOnly = true }
}

// WithTitle は画面上部の見出しを世界から導く関数を設定する。燃料メニューで残燃料を出し、投入で即座に
// 反映を見せる用途で使う。毎フレーム呼ぶので純粋な読み取りにする
func WithTitle(fn func(w.World) string) StorageOption {
	return func(st *StorageMenuState) { st.titleFunc = fn }
}

// State interface ================

var _ es.State[w.World] = &StorageMenuState{}
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
	storeTab := storageTabData{ID: tabIDStore, Label: query.T(world, "Store"), Items: st.toStorageItemData(world, st.filterStacks(world, query.BackpackStacks(world, player)))}
	// 燃料は入れたら即残数になり取り出さないので、投入タブだけにする
	if st.storeOnly {
		return StorageProps{Tabs: []storageTabData{storeTab}}, nil
	}
	retrieveTab := storageTabData{ID: tabIDRetrieve, Label: query.T(world, "Retrieve"), Items: st.toStorageItemData(world, st.filterStacks(world, query.StorageStacks(world, st.storageEntity)))}
	return StorageProps{Tabs: []storageTabData{retrieveTab, storeTab}}, nil
}

// filterStacks は両タブに出す束を itemFilter で絞る。フィルタ未指定なら素通しする
func (st *StorageMenuState) filterStacks(world w.World, stacks []query.Stack) []query.Stack {
	if st.itemFilter == nil {
		return stacks
	}
	filtered := make([]query.Stack, 0, len(stacks))
	for _, stack := range stacks {
		if st.itemFilter(world, stack.Rep) {
			filtered = append(filtered, stack)
		}
	}
	return filtered
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
		// 熱量列は燃料メニューでだけ出す。重量と同じく束の総量にする
		var heat string
		if st.showHeat {
			heat = (query.HeatContent(world, rep) * consts.Heat(stack.Count)).String()
		}
		items[i] = itemRowData{
			Entity: rep,
			Name:   query.GetEntityName(rep, world),
			Weight: total.KgString(),
			Heat:   heat,
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
		// 表示は filterStacks で絞り済みだが、投入実行でもフィルタを再確認して受け入れ品目を守る
		if st.itemFilter != nil && !st.itemFilter(world, item.Entity) {
			return nil
		}
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

// ViewUI はカテゴリタブとアイコン付きアイテム一覧を組む。
// 詳細モーダルは ScreenRenderer として Screen が本体の上へ重ねる。
func (st *StorageMenuState) ViewUI(world w.World, props StorageProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	content, pager := st.buildActiveListUI(world, props, cursor.TabIndex, cursor.ItemIndex, cursor.PageSize, res)
	title := ""
	if st.titleFunc != nil {
		title = st.titleFunc(world)
	}
	// タブが1つのときは選択中のタブ表示が浮くので、タブ無しのパネルで出す
	if len(props.Tabs) == 1 {
		return menuframe.PanelScreen(world, res, title, content, keybind.HelpHint(world), pager)
	}
	return menuframe.TabScreen(world, res, title, labels, cursor.TabIndex, content, keybind.HelpHint(world), pager)
}

// buildActiveListUI は行列とフッタ右端のページ表示を返す。
func (st *StorageMenuState) buildActiveListUI(world w.World, props StorageProps, tabIndex, itemIndex, perPage int, res resources.UIResources) ([]uicore.Drawable, string) {
	if tabIndex >= len(props.Tabs) {
		return nil, ""
	}
	currentTab := props.Tabs[tabIndex]
	// 熱量列を出すときはアイコン・名前の後ろに熱量・重量の2数値列、出さないときは重量のみ。
	// 熱量と重量は右寄せの数値どうしで隣接すると詰まって見えるので、間に空の間隔列を1つ挟む。
	// 列とセルは同じ順序で組み、片方だけずれる不整合を避ける
	cols := itemMenuColumns(styled.Num())
	if st.showHeat {
		cols = itemMenuColumns(styled.Num(), styled.Fit(), styled.Num())
	}
	rows := make([]menuframe.Row, len(currentTab.Items))
	for i, it := range currentTab.Items {
		if st.showHeat {
			rows[i] = itemMenuRow(world, it.Entity, it.Count, it.Heat, "", it.Weight)
		} else {
			rows[i] = itemMenuRow(world, it.Entity, it.Count, it.Weight)
		}
	}
	return menuframe.RenderList(itemIndex, rows, cols, menuframe.ListOpts{EmptyText: query.T(world, "No items"), ItemsPerPage: perPage}, res)
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
