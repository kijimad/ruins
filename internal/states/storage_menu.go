package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
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
	detail        menuscreen.Detail // 詳細モーダル。overlay として Screen に登録する
	screen        *menurt.Screen[StorageProps]
}

// State interface ================

var _ es.State[w.World] = &StorageMenuState{}
var _ menurt.ExtraInput = &StorageMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *StorageMenuState) OnStart(_ w.World) error {
	st.detail = menuscreen.NewEntityDetail(st.selectedEntity)
	st.screen = menurt.NewScreen[StorageProps](st, &st.detail)
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

// ExtraInput は共通入力に加える独自キーを返す。x で選択中の詳細モーダルを開く
func (st *StorageMenuState) ExtraInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return "", false
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
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("storageMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// StorageProps は画面の表示 props。menurt.Screen の型引数として渡す
type StorageProps struct {
	Tabs []storageTabData
}

type storageTabData struct {
	ID    tabID
	Label string
	Items []storageItemData
}

type storageItemData struct {
	Entity ecs.Entity
	Name   string
	Weight string
	Count  int
}

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *StorageMenuState) Fetch(world w.World) StorageProps {
	return StorageProps{
		Tabs: []storageTabData{
			{ID: tabIDRetrieve, Label: query.T(world, "Retrieve"), Items: st.createStorageItemData(world)},
			{ID: tabIDStore, Label: query.T(world, "Store"), Items: st.createBackpackItemData(world)},
		},
	}
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *StorageMenuState) Menu(props StorageProps) menurt.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menurt.MenuConfig{Key: "storage", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage}
}

func (st *StorageMenuState) createStorageItemData(world w.World) []storageItemData {
	items := query.GetStorageItems(world, st.storageEntity)
	sorted := query.SortEntities(world, items)
	return st.toStorageItemData(world, sorted)
}

func (st *StorageMenuState) createBackpackItemData(world w.World) []storageItemData {
	var entities []ecs.Entity
	backpackQuery := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for backpackQuery.Next() {
		entity := backpackQuery.Entity()
		entities = append(entities, entity)
	}

	sorted := query.SortEntities(world, entities)
	return st.toStorageItemData(world, sorted)
}

func (st *StorageMenuState) toStorageItemData(world w.World, entities []ecs.Entity) []storageItemData {
	items := make([]storageItemData, len(entities))
	for i, entity := range entities {
		name := query.GetEntityName(entity, world)
		item := storageItemData{
			Entity: entity,
			Name:   name,
			Weight: query.GetEntityWeight(world, entity).KgString(),
		}
		if world.Components.Stackable.Has(entity) {
			item.Count = world.Components.Stackable.Get(entity).Count
		}
		items[i] = item
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
		// 収納からバックパックへ移動
		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		if err := lifecycle.MoveToBackpack(world, item.Entity, playerEntity); err != nil {
			return err
		}
	case tabIDStore:
		// バックパックから収納へ移動
		if !query.CanAddToStorage(world, st.storageEntity, item.Entity) {
			return nil // 重量超過の場合は何もしない
		}
		if err := lifecycle.MoveToStorage(world, item.Entity, st.storageEntity); err != nil {
			return err
		}
	}

	return nil
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *StorageMenuState) View(world w.World, props StorageProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	// カテゴリをタブ帯に寄せ、本体は1カラムの一覧にする。性能は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	return newTabScreenUI(res, tabScreen{
		TabLabels: labels,
		TabIndex:  cursor.TabIndex,
		Content:   st.buildActiveListContainer(world, props, cursor.TabIndex, cursor.ItemIndex, res),
		Footer:    menuNavHint(world, true, query.T(world, "x Details")),
	})
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
	columnWidths := []int{260, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(currentTab.Items))
	for i, it := range currentTab.Items {
		rows[i] = menuRow{Cells: []string{nameWithCount(it.Name, it.Count), it.Weight}}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No items")}, res)
}
